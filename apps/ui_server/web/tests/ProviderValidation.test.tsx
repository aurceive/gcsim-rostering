import { act, renderHook, waitFor } from "@testing-library/react";
import type React from "react";
import { describe, expect, it } from "vitest";
import {
	ExecutorProvider,
	useExecutor,
	useRunResult,
} from "./ExecutorProvider";
import { makeExecutor, parsedResult } from "./testExecutor";

describe("ui_server validation errors", () => {
	it("shows errors returned by the engine before a run", async () => {
		const fake = makeExecutor({
			validate: () => Promise.resolve(parsedResult(["amber"], ["bad action"])),
		});
		const { result } = renderHook(
			() => ({ executor: useExecutor(), run: useRunResult() }),
			{
				wrapper: ({ children }: { children: React.ReactNode }) => (
					<ExecutorProvider exec={fake.supplier}>{children}</ExecutorProvider>
				),
			},
		);
		act(() => result.current.executor.run("config"));
		await waitFor(() => expect(result.current.run.error).toBe("bad action"));
		expect(fake.run).not.toHaveBeenCalled();
	});

	it("shows communication errors before a run", async () => {
		const fake = makeExecutor({
			validate: () => Promise.reject(new Error("network failed")),
		});
		const { result } = renderHook(
			() => ({ executor: useExecutor(), run: useRunResult() }),
			{
				wrapper: ({ children }: { children: React.ReactNode }) => (
					<ExecutorProvider exec={fake.supplier}>{children}</ExecutorProvider>
				),
			},
		);
		act(() => result.current.executor.run("config"));
		await waitFor(() =>
			expect(result.current.run.error).toContain("network failed"),
		);
		expect(fake.run).not.toHaveBeenCalled();
	});
});
