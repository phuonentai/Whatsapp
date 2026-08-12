import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import { renderWithProviders } from "@/test/render";
import { DocumentSources } from "./document-sources";
import { useDocuments } from "@/lib/hooks/queries/use-documents-query";
import type { SimilarDocument } from "@/lib/models/cognitive.model";
import type { Document } from "@/lib/models/document.model";

vi.mock("@/lib/hooks/queries/use-documents-query", () => ({
  useDocuments: vi.fn(),
}));

const mockUseDocuments = vi.mocked(useDocuments);

function makeDocument(overrides: Partial<Document>): Document {
  return {
    id: 1,
    title: "Manual de productos",
    fileName: "manual.pdf",
    contentType: "application/pdf",
    fileSize: 1024,
    status: "processed",
    visibility: "workspace",
    createdAt: new Date("2026-01-01T00:00:00Z"),
    updatedAt: new Date("2026-01-01T00:00:00Z"),
    ...overrides,
  };
}

function makeSource(overrides: Partial<SimilarDocument>): SimilarDocument {
  return {
    id: 10,
    documentId: 1,
    contentPreview: "Fragmento citado del documento…",
    similarityScore: 0.87,
    ...overrides,
  };
}

describe("DocumentSources", () => {
  beforeEach(() => {
    mockUseDocuments.mockReset();
  });

  it("joins document titles from the documents query", async () => {
    mockUseDocuments.mockReturnValue([
      makeDocument({ id: 1, title: "Manual de productos", fileName: "manual.pdf" }),
      makeDocument({ id: 2, title: "Políticas de envío", fileName: "politicas.xlsx", contentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" }),
    ]);

    renderWithProviders(
      <DocumentSources
        sources={[makeSource({ id: 11, documentId: 1 }), makeSource({ id: 12, documentId: 2, similarityScore: 0.62 })]}
      />
    );

    fireEvent.click(screen.getByRole("button", { name: /2 fuentes consultadas/i }));

    expect(screen.getByText("Manual de productos")).toBeDefined();
    expect(screen.getByText("Políticas de envío")).toBeDefined();
    expect(screen.queryByText(/Document #/)).toBeNull();
  });

  it("falls back to 'Documento no disponible' for unknown document ids", async () => {
    mockUseDocuments.mockReturnValue([makeDocument({ id: 1 })]);

    renderWithProviders(
      <DocumentSources sources={[makeSource({ id: 11, documentId: 1 }), makeSource({ id: 13, documentId: 999 })]} />
    );

    fireEvent.click(screen.getByRole("button", { name: /2 fuentes consultadas/i }));

    expect(screen.getByText("Manual de productos")).toBeDefined();
    expect(screen.getByText("Documento no disponible")).toBeDefined();
  });

  it("renders a single-source label in singular form", () => {
    mockUseDocuments.mockReturnValue([makeDocument({ id: 1 })]);

    renderWithProviders(<DocumentSources sources={[makeSource({ id: 11, documentId: 1 })]} />);

    expect(screen.getByRole("button", { name: /1 fuente consultada/i })).toBeDefined();
  });
});
