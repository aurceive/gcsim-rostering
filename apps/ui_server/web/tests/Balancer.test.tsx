import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useBalancer } from "../../../kqmsim/src/Balancer";

vi.mock("react-i18next", async (importOriginal) => ({
	...await importOriginal<typeof import("react-i18next")>(),
	useTranslation: () => ({ i18n: { resolvedLanguage: "en" } }),
}));

const encode = (event: unknown) =>
	new TextEncoder().encode(`${JSON.stringify(event)}\n`);

function response() {
	let stream!: ReadableStreamDefaultController<Uint8Array>;
	const body = new ReadableStream<Uint8Array>({
		start(controller) {
			stream = controller;
		},
	});
	vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(body)));
	return stream;
}

afterEach(() => {
	cleanup();
	vi.unstubAllGlobals();
});

describe("ui_server balancer", () => {
	it("streams logs, then applies the result and hides the panel", async () => {
		const stream = response();
		const apply = vi.fn().mockReturnValue(true);
		const { result } = renderHook(() => useBalancer("gcsim", "original", apply));
		act(() => { result.current.button.props.onClick(); });
		await act(async () => {
			stream.enqueue(encode({ type: "start" }));
			stream.enqueue(encode({ type: "log", data: btoa("progress") }));
		});
		await waitFor(() => expect(result.current.running).toBe(true));
		expect(result.current.panel).not.toBeNull();
		await waitFor(() => expect(result.current.panel?.props.children.props.children[0]).toBe("progress"));
		await act(async () => {
			stream.enqueue(encode({ type: "result", text: "balanced" }));
			stream.close();
		});
		await waitFor(() => expect(result.current.panel).toBeNull());
		expect(apply).toHaveBeenCalledWith("original", "balanced");
		expect(result.current.running).toBe(false);
		expect(fetch).toHaveBeenCalledWith("/api/engines/gcsim/balance", expect.objectContaining({ method: "POST" }));
	});

	it("keeps the panel on an engine error and never applies the config", async () => {
		const stream = response();
		const apply = vi.fn();
		const { result } = renderHook(() => useBalancer("gcsim", "original", apply));
		act(() => { result.current.button.props.onClick(); });
		await act(async () => {
			stream.enqueue(encode({ type: "error", text: "bad config" }));
			stream.close();
		});
		await waitFor(() => expect(result.current.panel?.props.role).toBe("alert"));
		expect(result.current.running).toBe(false);
		expect(apply).not.toHaveBeenCalled();
	});

	it("keeps the current config when the original has changed", async () => {
		const stream = response();
		const apply = vi.fn().mockReturnValue(false);
		const { result } = renderHook(() => useBalancer("gcsim", "original", apply));
		act(() => { result.current.button.props.onClick(); });
		await act(async () => {
			stream.enqueue(encode({ type: "result", text: "balanced" }));
			stream.close();
		});
		await waitFor(() => expect(result.current.panel?.props.role).toBe("alert"));
		expect(apply).toHaveBeenCalledWith("original", "balanced");
	});

	it("reports a truncated stream", async () => {
		const stream = response();
		const apply = vi.fn();
		const { result } = renderHook(() => useBalancer("gcsim", "original", apply));
		act(() => { result.current.button.props.onClick(); });
		await act(async () => { stream.close(); });
		await waitFor(() => expect(result.current.panel?.props.role).toBe("alert"));
		expect(apply).not.toHaveBeenCalled();
	});

	it("preserves UTF-8 characters split across log chunks", async () => {
		const stream = response();
		const { result } = renderHook(() => useBalancer("gcsim", "original", () => true));
		act(() => { result.current.button.props.onClick(); });
		await act(async () => {
			stream.enqueue(encode({ type: "log", data: btoa(String.fromCharCode(0xd0)) }));
			stream.enqueue(encode({ type: "log", data: btoa(String.fromCharCode(0x9f)) }));
		});
		await waitFor(() => expect(result.current.panel?.props.children.props.children[0]).toBe("П"));
		await act(async () => {
			stream.enqueue(encode({ type: "error", text: "failed" }));
			stream.close();
		});
	});
});
