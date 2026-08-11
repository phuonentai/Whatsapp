import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, fireEvent, screen } from "@testing-library/react";

import { useGlobalShortcuts, isTypingTarget } from "./use-global-shortcuts";
import { useCommandPaletteStore } from "@/lib/stores/command-palette-store";
import { useShortcutsHelpStore } from "@/lib/stores/shortcuts-help-store";

const pushMock = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: pushMock }),
}));

function Harness() {
  useGlobalShortcuts();
  return (
    <>
      <input aria-label="search input" />
      <textarea aria-label="note input" />
      <div aria-label="editable" contentEditable suppressContentEditableWarning />
    </>
  );
}

describe("isTypingTarget", () => {
  it("returns true for input, textarea, and contenteditable", () => {
    const input = document.createElement("input");
    const textarea = document.createElement("textarea");
    const editable = document.createElement("div");
    editable.setAttribute("contenteditable", "true");

    expect(isTypingTarget(input)).toBe(true);
    expect(isTypingTarget(textarea)).toBe(true);
    expect(isTypingTarget(editable)).toBe(true);
    expect(isTypingTarget(document.body)).toBe(false);
    expect(isTypingTarget(null)).toBe(false);
  });
});

describe("useGlobalShortcuts", () => {
  beforeEach(() => {
    pushMock.mockReset();
    useCommandPaletteStore.getState().closePalette();
    useShortcutsHelpStore.getState().closeHelp();
  });

  it("navigates on g then d", () => {
    render(<Harness />);

    fireEvent.keyDown(window, { key: "g" });
    fireEvent.keyDown(window, { key: "d" });

    expect(pushMock).toHaveBeenCalledWith("/dashboard");
  });

  it("navigates on g then i to inbox", () => {
    render(<Harness />);

    fireEvent.keyDown(window, { key: "g" });
    fireEvent.keyDown(window, { key: "i" });

    expect(pushMock).toHaveBeenCalledWith("/dashboard/inbox");
  });

  it("opens the command palette on Cmd+K", () => {
    render(<Harness />);

    fireEvent.keyDown(window, { key: "k", metaKey: true });

    expect(useCommandPaletteStore.getState().open).toBe(true);
  });

  it("opens the shortcuts help overlay on question mark", () => {
    render(<Harness />);

    fireEvent.keyDown(window, { key: "?" });

    expect(useShortcutsHelpStore.getState().open).toBe(true);
  });

  it("suppresses g-navigation while typing in an input", () => {
    render(<Harness />);

    const input = screen.getByLabelText("search input");
    fireEvent.keyDown(input, { key: "g" });
    fireEvent.keyDown(input, { key: "d" });

    expect(pushMock).not.toHaveBeenCalled();
  });

  it("suppresses g-navigation while typing in a contenteditable", () => {
    render(<Harness />);

    const editable = screen.getByLabelText("editable");
    fireEvent.keyDown(editable, { key: "g" });
    fireEvent.keyDown(editable, { key: "c" });

    expect(pushMock).not.toHaveBeenCalled();
  });

  it("still honors Cmd+K while typing in an input", () => {
    render(<Harness />);

    const input = screen.getByLabelText("search input");
    fireEvent.keyDown(input, { key: "k", metaKey: true });

    expect(useCommandPaletteStore.getState().open).toBe(true);
  });
});
