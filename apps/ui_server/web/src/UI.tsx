import {
	DefaultSampleOptions,
	defaultEditorPrefs,
	Editor,
	ExecutorProvider,
	namedSeeds,
	ResultsView,
	ResultWarnings,
	SampleLog,
	SeedPicker,
	useExecutor,
	useRunResult,
	useValidation,
} from "@gcsim/components";
import { dynamicKey } from "@gcsim/localization";
import {
	Button,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@gcsim/primitives";
import type { Executor, ExecutorSupplier, model, Sample } from "@gcsim/types";
import { useLocalStorage } from "@gcsim/utils";
import { type ReactNode, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { teamCharacters } from "./characters";
import { useBalancer } from "./Balancer";
import { downloadSample } from "./downloadSample";

export function UI({
	exec,
	config,
	setConfig,
	children,
	visible = true,
	engine,
	applyBalancedConfig,
}: {
	exec: ExecutorSupplier<Executor>;
	config?: string;
	setConfig?: (value: string) => void;
	children?: ReactNode;
	visible?: boolean;
	engine?: string;
	applyBalancedConfig?: (original: string, value: string) => boolean;
	gitCommit?: string;
	mode?: string;
}) {
	const [localConfig, setLocalConfig] = useLocalStorage("cfg", "");
	return (
		<ExecutorProvider exec={exec}>
			<Workspace
				config={config ?? localConfig}
				setConfig={setConfig ?? setLocalConfig}
				settings={children}
				visible={visible}
				engine={engine}
				applyBalancedConfig={applyBalancedConfig}
			/>
		</ExecutorProvider>
	);
}

function Workspace({
	config,
	setConfig,
	settings,
	visible,
	engine,
	applyBalancedConfig,
}: {
	config: string;
	setConfig: (value: string) => void;
	settings?: ReactNode;
	visible: boolean;
	engine?: string;
	applyBalancedConfig?: (original: string, value: string) => boolean;
}) {
	const { t } = useTranslation();
	const { exec, isReady, busy, run, cancel } = useExecutor();
	const { result, error } = useRunResult();
	const [incomingConfig, setIncomingConfig] = useState(() => {
		const value = new URLSearchParams(window.location.hash.slice(1)).get(
			"config",
		);
		return value?.trim() ? value : null;
	});
	const [prefs, setPrefs] = useLocalStorage(
		"kqm-editor-prefs",
		defaultEditorPrefs,
	);
	const [tab, setTab] = useState("simulator");
	const [settingsOpen, setSettingsOpen] = useState(false);
	const [loaded, setLoaded] = useState<model.SimulationResult | null>(null);
	const [loadError, setLoadError] = useState("");
	const [loading, setLoading] = useState(false);
	const { isValid, error: configError, parsedTeam } = useValidation(config);
	const [copied, setCopied] = useState(false);
	const [copyError, setCopyError] = useState("");
	const [sample, setSample] = useState<Sample | null>(null);
	const [seed, setSeed] = useState<string | null>(null);
	const [sampleError, setSampleError] = useState("");
	const [generating, setGenerating] = useState(false);
	const [sampleSettings, setSampleSettings] = useLocalStorage<string[]>(
		"kqm-sample-settings",
		DefaultSampleOptions,
	);
	const data = loaded ?? result;
	const balancer = useBalancer(engine ?? "", config, applyBalancedConfig ?? (() => false));

	function dismissConfigImport() {
		setIncomingConfig(null);
		window.history.replaceState(
			window.history.state,
			"",
			window.location.pathname + window.location.search,
		);
	}

	useEffect(() => {
		const match = window.location.pathname.match(
			/^\/sh\/([A-Za-z0-9_-]{1,150})$/,
		);
		if (!match) return;
		const controller = new AbortController();
		setLoading(true);
		setTab("results");
		fetch(`/api/share/${encodeURIComponent(match[1])}`, {
			signal: controller.signal,
		})
			.then(async (response) => {
				if (!response.ok)
					throw new Error(
						`The shared result could not be loaded (HTTP ${response.status}).`,
					);
				return response.json();
			})
			.then((value: model.SimulationResult) => {
				if (!controller.signal.aborted) setLoaded(value);
			})
			.catch((reason: unknown) => {
				if (!controller.signal.aborted)
					setLoadError(
						reason instanceof Error ? reason.message : String(reason),
					);
			})
			.finally(() => {
				if (!controller.signal.aborted) setLoading(false);
			});
		return () => controller.abort();
	}, []);

	function start() {
		setLoaded(null);
		setLoadError("");
		setCopied(false);
		setCopyError("");
		setSample(null);
		setSeed(null);
		setSampleError("");
		if (window.location.pathname.startsWith("/sh/"))
			window.history.replaceState(null, "", "/");
		run(config);
		setTab("results");
	}

	async function generateSample(next: string) {
		if (!data?.config_file || busy) return;
		setGenerating(true);
		setSampleError("");
		setSeed(next);
		try {
			setSample(await exec().sample(data.config_file, next));
		} catch (reason) {
			setSampleError(String(reason));
		} finally {
			setGenerating(false);
		}
	}

	if (!visible) return null;

	return (
		<main className="kqm-main">
			<div className="kqm-tabs" role="tablist" aria-label="Simulation">
				{["simulator", "results", "sample"].map((value) => (
					<Button
						key={value}
						role="tab"
						aria-selected={tab === value}
						variant={tab === value ? "default" : "outline"}
						disabled={value !== "simulator" && !data && !loading}
						onClick={() => setTab(value)}
					>
						{value[0].toUpperCase() + value.slice(1)}
					</Button>
				))}
			</div>
			{(error || loadError) && (
				<div role="alert" className="kqm-error">
					{error || loadError}
					<Button
						variant="outline"
						onClick={() => {
							cancel();
							setTab("simulator");
						}}
					>
						Return to simulator
					</Button>
				</div>
			)}
			{tab === "simulator" && (
				<Editor
					config={config}
					setConfig={setConfig}
					parsedTeam={parsedTeam}
					teamCharacters={teamCharacters}
					error={configError}
					prefs={prefs}
					onPrefsChange={setPrefs}
					showThemeSelector
					canRun={isReady && isValid && !busy && !balancer.running}
					busy={!isReady || busy}
					onRun={start}
					extraAction={engine && applyBalancedConfig ? balancer.button : null}
					actionPanel={engine && applyBalancedConfig ? balancer.panel : null}
					settings={
						<Button variant="outline" onClick={() => setSettingsOpen(true)}>
							Settings
						</Button>
					}
				/>
			)}
			{tab === "results" && (
				<>
					{loading && <p role="status">Loading simulation results...</p>}
					{busy && (
						<div role="status" className="kqm-actions">
							Running: {data?.statistics?.iterations ?? 0} /{" "}
							{data?.simulator_settings?.iterations ?? "..."}
							<Button onClick={cancel}>Stop</Button>
						</div>
					)}
					{data && (
						<>
							{!busy && <ResultWarnings data={data} />}
							<div className="kqm-actions">
								<Button
									variant="outline"
									onClick={async () => {
										try {
											await navigator.clipboard.writeText(
												data.config_file ?? "",
											);
											setCopied(true);
											setCopyError("");
										} catch (reason) {
											setCopied(false);
											setCopyError(`Cannot copy config: ${String(reason)}`);
										}
									}}
								>
									{copied ? "Copied" : "Copy config"}
								</Button>
								<Button
									variant="outline"
									onClick={() => {
										setConfig(data.config_file ?? "");
										setTab("simulator");
									}}
								>
									Edit config
								</Button>
							</div>
							{copyError && <p role="alert">{copyError}</p>}
							<ResultsView
								model={data}
								names={data.character_details?.map((c) =>
									t(dynamicKey(`game:character_names.${c.name}`)),
								)}
							/>
						</>
					)}
				</>
			)}
			{tab === "sample" && data && (
				<>
					<SeedPicker
						seeds={namedSeeds(data)}
						value={seed}
						onPick={generateSample}
						running={busy || generating || !isReady}
					/>
					{generating && <p role="status">Generating sample...</p>}
					{sampleError && <p role="alert">{sampleError}</p>}
					{sample && (
						<SampleLog
							sample={sample}
							settings={sampleSettings}
							onSettingsChange={setSampleSettings}
							onDownload={downloadSample}
						/>
					)}
				</>
			)}
			<Dialog
				open={incomingConfig !== null}
				onOpenChange={(open) => {
					if (!open) dismissConfigImport();
				}}
			>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Load database config?</DialogTitle>
						<DialogDescription>
							{config.trim()
								? "This will replace your saved config."
								: "Load this config from the database."}
						</DialogDescription>
					</DialogHeader>
					<DialogFooter>
						<Button variant="outline" onClick={dismissConfigImport}>
							Cancel
						</Button>
						<Button
							onClick={() => {
								if (incomingConfig === null) return;
								setConfig(incomingConfig);
								setTab("simulator");
								dismissConfigImport();
							}}
						>
							{config.trim() ? "Replace config" : "Load config"}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
			<Dialog open={settingsOpen} onOpenChange={setSettingsOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Settings</DialogTitle>
					</DialogHeader>
					{settings ?? <p>Engine selection is available in the header.</p>}
				</DialogContent>
			</Dialog>
		</main>
	);
}
