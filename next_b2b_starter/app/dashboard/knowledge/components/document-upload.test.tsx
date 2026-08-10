import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/render";
import { DocumentUpload } from "./document-upload";

function makeFile(name: string, type: string) {
  return new File(["content"], name, { type });
}

/** Drive react-dropzone's input directly, bypassing the accept-attribute check
 * that both userEvent.upload and the browser file picker enforce. */
function uploadFileToInput(input: HTMLInputElement, file: File) {
  Object.defineProperty(input, "files", { value: [file], configurable: true });
  fireEvent.change(input);
}

describe("DocumentUpload", () => {
  const onUpload = vi.fn();

  beforeEach(() => {
    onUpload.mockReset();
    onUpload.mockResolvedValue(undefined);
  });

  it("accepts a PDF, shows the title form and uploads it", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(<DocumentUpload onUpload={onUpload} />);
    const input = container.querySelector('input[type="file"]') as HTMLInputElement;
    expect(input).not.toBeNull();

    await user.upload(input, makeFile("factura.pdf", "application/pdf"));

    const titleInput = screen.getByPlaceholderText("My Document");
    expect(titleInput).toBeInTheDocument();
    await user.clear(titleInput);
    await user.type(titleInput, "Facturas 2026");
    await user.click(screen.getByRole("button", { name: "Upload PDF" }));

    expect(onUpload).toHaveBeenCalledTimes(1);
    const [file, title] = onUpload.mock.calls[0];
    expect(file.name).toBe("factura.pdf");
    expect(title).toBe("Facturas 2026");
  });

  it("rejects a non-PDF file with an error and does not upload", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(<DocumentUpload onUpload={onUpload} />);
    const input = container.querySelector('input[type="file"]') as HTMLInputElement;

    uploadFileToInput(input, makeFile("notas.txt", "text/plain"));

    expect(await screen.findByText("Only PDF files are accepted")).toBeInTheDocument();
    expect(screen.queryByText("Upload PDF")).not.toBeInTheDocument();
    expect(onUpload).not.toHaveBeenCalled();
  });

  it("does not upload while no file is selected", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DocumentUpload onUpload={onUpload} />);
    expect(screen.queryByRole("button", { name: "Upload PDF" })).not.toBeInTheDocument();
    await user.click(screen.getByText("Click or drag PDF to upload"));
    expect(onUpload).not.toHaveBeenCalled();
  });
});
