import { describe, it, expect, vi, afterEach } from "vitest";
import { screen, render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Markdown } from "./markdown";

describe("Markdown", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("escapes raw HTML tags and never renders a script element", () => {
    const { container } = render(
      <Markdown content={'<script>alert("xss")</script>\n\n<img src=x onerror=alert(1)>'} />
    );

    expect(container.querySelector("script")).toBeNull();
    expect(container.querySelector("img")).toBeNull();
    // The raw source is visible as escaped text.
    expect(screen.getByText(/<script>alert\("xss"\)<\/script>/)).toBeDefined();
    expect(screen.getByText(/<img src=x onerror=alert\(1\)>/)).toBeDefined();
  });

  it("renders markdown structure (headings, lists, emphasis, links)", () => {
    render(
      <Markdown
        content={
          "# Título\n\nTexto con **negrita** y *cursiva*.\n\n- ítem uno\n- ítem dos\n\n[enlace](https://example.com)"
        }
      />
    );

    expect(screen.getByRole("heading", { level: 1, name: "Título" })).toBeDefined();
    expect(screen.getByText(/Texto con/)).toBeDefined();
    expect(screen.getByText("negrita")).toHaveProperty("tagName", "STRONG");
    expect(screen.getByText("cursiva")).toHaveProperty("tagName", "EM");
    expect(screen.getByRole("list")).toBeDefined();
    expect(screen.getAllByRole("listitem")).toHaveLength(2);
    const link = screen.getByRole("link", { name: "enlace" });
    expect(link.getAttribute("href")).toBe("https://example.com");
    expect(link.getAttribute("rel")).toContain("noopener");
  });

  it("renders GFM tables and task lists", () => {
    render(
      <Markdown
        content={
          "| A | B |\n|---|---|\n| 1 | 2 |\n\n- [x] hecho\n- [ ] pendiente"
        }
      />
    );

    expect(screen.getByRole("table")).toBeDefined();
    expect(screen.getByRole("columnheader", { name: "A" })).toBeDefined();
    expect(screen.getByText("hecho")).toBeDefined();
    expect(screen.getByText("pendiente")).toBeDefined();
  });

  it("shows a copy button only when requested and copies the raw content", async () => {
    const user = userEvent.setup();
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText },
      configurable: true,
    });

    const { rerender } = render(<Markdown content="Hola **mundo**" />);
    expect(screen.queryByRole("button", { name: "Copiar" })).toBeNull();

    rerender(<Markdown content="Hola **mundo**" showCopyButton />);
    const copyButton = screen.getByRole("button", { name: "Copiar" });
    await user.click(copyButton);

    expect(writeText).toHaveBeenCalledWith("Hola **mundo**");
    expect(await screen.findByRole("button", { name: "Copiado" })).toBeDefined();
  });
});
