import { ServerExecutor, WasmExecutor } from "@gcsim/executors";
import { Toaster } from "@gcsim/primitives";
import { useLocalStorage } from "@gcsim/utils";
import React from "react";
import { UI } from "./UI";

type EngineStatus = {
	name: string;
	available: boolean;
	running: boolean;
	error?: string;
};

const selectionKey = "ui-server-engine";
const browserEngine = "kqmsim-browser";

async function responseError(response: Response): Promise<string> {
	return (await response.text()).trim() || `HTTP ${response.status}`;
}

function EngineWorkspace({
	name,
	visible,
	config,
	setConfig,
	applyBalancedConfig,
	status,
}: {
	name: string;
	visible: boolean;
	config: string;
	setConfig: (value: string) => void;
	applyBalancedConfig: (original: string, value: string) => boolean;
	status?: EngineStatus;
}) {
	const [started, setStarted] = React.useState(false);
	const [error, setError] = React.useState("");
	const [attempt, setAttempt] = React.useState(0);
	const [starting, setStarting] = React.useState(false);
	// biome-ignore lint/correctness/useExhaustiveDependencies: Retry needs a fresh executor and request ID.
	const supplier = React.useMemo(() => {
		const executor = new ServerExecutor(`/engine/${encodeURIComponent(name)}`);
		return () => executor;
	}, [name, attempt]);

	// biome-ignore lint/correctness/useExhaustiveDependencies: Retry must start the engine again.
	React.useEffect(() => {
		let active = true;
		setStarting(true);
		setError("");
		fetch(`/api/engines/${encodeURIComponent(name)}/start`, { method: "POST" })
			.then(async (response) => {
				if (!response.ok) throw new Error(await responseError(response));
				if (active) setStarted(true);
			})
			.catch((reason: unknown) => {
				if (active)
					setError(reason instanceof Error ? reason.message : String(reason));
			})
			.finally(() => {
				if (active) setStarting(false);
			});
		return () => {
			active = false;
		};
	}, [name, attempt]);

	return (
		<div hidden={!visible}>
			{starting && <p role="status">Starting {name}...</p>}
			{(error || (!starting && !status?.running && status?.error)) && (
				<div className="kqm-error" role="alert">
					{error || status?.error}{" "}
					<button
						type="button"
						onClick={() => {
							setStarted(false);
							setAttempt((value) => value + 1);
						}}
					>
						Retry
					</button>
				</div>
			)}
			{started && (
				<UI
					exec={supplier}
					config={config}
					setConfig={setConfig}
					applyBalancedConfig={applyBalancedConfig}
					engine={name}
					visible={visible}
				/>
			)}
		</div>
	);
}

function BrowserWorkspace({
	visible,
	config,
	setConfig,
	applyBalancedConfig,
}: {
	visible: boolean;
	config: string;
	setConfig: (value: string) => void;
	applyBalancedConfig: (original: string, value: string) => boolean;
}) {
	const [workers, setWorkers] = useLocalStorage("wasm-num-workers", 3);
	const executor = React.useMemo(() => new WasmExecutor("/main.wasm"), []);
	const supplier = React.useCallback(() => executor, [executor]);
	React.useEffect(() => {
		executor.setWorkerCount(workers);
	}, [executor, workers]);
	return (
		<div hidden={!visible}>
			<UI
				exec={supplier}
				config={config}
				setConfig={setConfig}
				applyBalancedConfig={applyBalancedConfig}
				engine="kqmsim"
				visible={visible}
			>
				<label>
					Browser workers{" "}
					<input
						type="number"
						min={1}
						max={30}
						value={workers}
						onChange={(event) => {
							const value = Number(event.target.value);
							if (Number.isInteger(value) && value >= 1 && value <= 30)
								setWorkers(value);
						}}
					/>
				</label>
			</UI>
		</div>
	);
}

export default function App() {
	const [engine, setEngine] = useLocalStorage(selectionKey, "gcsim");
	const [config, setConfig] = useLocalStorage("cfg", "");
	const configRef = React.useRef(config);
	configRef.current = config;
	const updateConfig = (value: string) => {
		configRef.current = value;
		setConfig(value);
	};
	const applyBalancedConfig = (original: string, value: string) => {
		if (configRef.current !== original) return false;
		updateConfig(value);
		return true;
	};
	const [visited, setVisited] = React.useState<string[]>([]);
	const [engines, setEngines] = React.useState<EngineStatus[]>([]);
	const [loadError, setLoadError] = React.useState("");

	React.useEffect(() => {
		let active = true;
		const update = () => {
			fetch("/api/engines")
				.then(async (response) => {
					if (!response.ok) throw new Error(await responseError(response));
					return (await response.json()) as EngineStatus[];
				})
				.then((items) => {
					if (!active) return;
					setEngines(items);
					setLoadError("");
				})
				.catch((reason: unknown) => {
					if (active)
						setLoadError(
							reason instanceof Error ? reason.message : String(reason),
						);
				});
		};
		update();
		const timer = window.setInterval(update, 3000);
		return () => {
			active = false;
			window.clearInterval(timer);
		};
	}, []);

	React.useEffect(() => {
		if (
			engine === browserEngine ||
			engines.some((item) => item.name === engine)
		) {
			setVisited((items) =>
				items.includes(engine) ? items : [...items, engine],
			);
		}
	}, [engine, engines]);

	const selected = engines.find((item) => item.name === engine);
	const recognized = selected || engine === browserEngine;
	return (
		<>
			<Toaster position="top-right" theme="dark" />
			<header className="kqm-header">
				<a href="https://keqingmains.com">
					<img src="/kqm-logo.png" alt="KQM" width="48" height="48" />
				</a>
				<span className="kqm-title">ui_server</span>
				<label>
					Engine{" "}
					<select
						value={engine}
						onChange={(event) => setEngine(event.target.value)}
					>
						{!recognized && (
							<option value={engine}>{engine} (not configured)</option>
						)}
						{engines.map((item) => (
							<option value={item.name} key={item.name}>
								{item.name}
								{item.available ? "" : " (not built)"}
							</option>
						))}
						<option value={browserEngine}>kqmsim (browser/WASM)</option>
					</select>
				</label>
				<nav aria-label="KQM sites">
					<a href="https://db.kqm.gg">Database</a>
				</nav>
			</header>
			{loadError && (
				<p className="kqm-error" role="alert">
					Cannot load engines: {loadError}
				</p>
			)}
			{engines.length > 0 && !recognized && (
				<p className="kqm-error" role="alert">
					Saved engine "{engine}" is not configured. Select another engine.
				</p>
			)}
			{visited.map((name) =>
				name === browserEngine ? (
					<BrowserWorkspace
						key={name}
						visible={name === engine}
						config={config}
						setConfig={updateConfig}
						applyBalancedConfig={applyBalancedConfig}
					/>
				) : (
					<EngineWorkspace
						key={name}
						name={name}
						visible={name === engine}
						config={config}
						setConfig={updateConfig}
						applyBalancedConfig={applyBalancedConfig}
						status={engines.find((item) => item.name === name)}
					/>
				),
			)}
			<footer className="kqm-footer">
				Based on KQM Sim and gcsim.{" "}
				<a href="https://github.com/KQM-git/kqmsim">
					KQM Sim source (AGPL-3.0)
				</a>
			</footer>
		</>
	);
}
