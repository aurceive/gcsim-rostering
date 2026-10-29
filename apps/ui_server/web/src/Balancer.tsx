import { Button, Spinner } from "@gcsim/primitives";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

type Event = { type: string; text?: string; data?: string };
const maxLog = 1024 * 1024;

export function useBalancer(
	engine: string,
	config: string,
	apply: (original: string, value: string) => boolean,
) {
	const { i18n } = useTranslation();
	const russian = i18n.resolvedLanguage === "ru";
	const label = russian ? "Балансер" : "Balancer";
	const [running, setRunning] = useState(false);
	const [log, setLog] = useState("");
	const [error, setError] = useState("");
	const [open, setOpen] = useState(false);
	const controller = useRef<AbortController | null>(null);
	const logPanel = useRef<HTMLPreElement>(null);

	useEffect(() => () => controller.current?.abort(), []);
	useEffect(() => {
		if (logPanel.current) logPanel.current.scrollTop = logPanel.current.scrollHeight;
	}, [log]);

	async function start() {
		if (controller.current) return;
		const request = new AbortController();
		controller.current = request;
		const original = config;
		setRunning(true);
		setLog("");
		setError("");
		setOpen(true);
		try {
			const response = await fetch(`/api/engines/${encodeURIComponent(engine)}/balance`, {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ config: original }),
				signal: request.signal,
			});
			if (!response.ok) throw new Error(await response.text());
			if (!response.body) throw new Error("Balancer response has no stream.");
			const reader = response.body.getReader();
			const decoder = new TextDecoder();
			const logDecoder = new TextDecoder();
			let pending = "";
			let result: string | null = null;
			let finished = false;
			const handle = (line: string) => {
				const event: Event = JSON.parse(line);
				if (finished) throw new Error("Unexpected data after balancer result.");
				if (event.type === "log") {
					if (typeof event.data !== "string") throw new Error("Invalid balancer log.");
					const bytes = Uint8Array.from(atob(event.data), (char) => char.charCodeAt(0));
					const text = logDecoder.decode(bytes, { stream: true });
					setLog((previous) => {
						const combined = previous + text;
						return combined.length > maxLog
							? `[Earlier output truncated]\n${combined.slice(-maxLog)}`
							: combined;
					});
				} else if (event.type === "error") {
					throw new Error(event.text || "Balancer failed.");
				} else if (event.type === "result") {
					if (!event.text?.trim()) throw new Error("Balancer returned an empty config.");
					result = event.text;
					finished = true;
				} else if (event.type !== "start") {
					throw new Error(`Unknown balancer event: ${event.type}`);
				}
			};
			while (true) {
				const chunk = await reader.read();
				pending += decoder.decode(chunk.value, { stream: !chunk.done });
				let end = pending.indexOf("\n");
				while (end !== -1) {
					handle(pending.slice(0, end));
					pending = pending.slice(end + 1);
					end = pending.indexOf("\n");
				}
				if (chunk.done) break;
			}
			if (pending.trim() || !finished || result === null)
				throw new Error("Balancer response ended before completion.");
			if (!apply(original, result))
				throw new Error(russian
					? "Конфиг изменился во время работы балансера. Текущий конфиг не заменён."
					: "The config changed while the balancer was running. The current config was not replaced.");
			setOpen(false);
		} catch (reason) {
			if (!request.signal.aborted) {
				setError(reason instanceof Error ? reason.message : String(reason));
				setRunning(false);
				request.abort();
			}
		} finally {
			if (controller.current === request) {
				controller.current = null;
				if (!request.signal.aborted) setRunning(false);
			}
		}
	}

	return {
		running,
		button: (
			<Button className="flex-1" onClick={start} disabled={running || !config.trim()}>
				{running ? <Spinner /> : (
					<svg aria-hidden="true" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
						<path d="M12 3v18M3 7h18M5 7 2 13h6L5 7Zm14 0-3 6h6l-3-6ZM7 21h10" />
					</svg>
				)}
				{label}
			</Button>
		),
		panel: open ? (
			<div
				className="basis-full rounded-g-md border bg-g-surface p-3"
				style={{ borderColor: error ? "#a55b5b" : undefined, flexBasis: "calc(100% + 16px)", margin: "0 -8px" }}
				role={error ? "alert" : "status"}
			>
				<pre
					ref={logPanel}
					style={{ height: "200px", overflow: "auto", whiteSpace: "pre-wrap", overflowWrap: "anywhere", margin: 0 }}
				>
					{log || (running ? (russian ? "Запуск балансера..." : "Starting balancer...") : "")}
					{error && `\n${error}`}
				</pre>
			</div>
		) : null,
	};
}
