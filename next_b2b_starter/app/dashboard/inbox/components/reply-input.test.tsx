import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { renderWithProviders } from "@/test/render";
import { ui } from "@/lib/copy/ui";
import { ReplyInput } from "./reply-input";

const mocks = vi.hoisted(() => ({
  rephrase: vi.fn(),
  pending: { value: false },
}));

vi.mock("@/lib/hooks/mutations/use-rephrase-mutation", () => ({
  useRephraseMutation: () => ({
    mutateAsync: mocks.rephrase,
    isPending: mocks.pending.value,
  }),
  isAiCreditsExhausted: (error: unknown) =>
    error instanceof Error && /API Error 402/.test(error.message),
}));

const sendButton = () => screen.getByRole("button", { name: "Enviar" });
const assistTrigger = () =>
  screen.getByRole("button", { name: ui.agent.rephraseTrigger });

describe("ReplyInput", () => {
  beforeEach(() => {
    mocks.rephrase.mockReset();
    mocks.pending.value = false;
  });

  it("disables send for empty/whitespace input", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn().mockResolvedValue(undefined);
    renderWithProviders(
      <ReplyInput onSend={onSend} isSending={false} conversationId={1} />
    );
    const input = screen.getByPlaceholderText("Escribe un mensaje...");
    expect(sendButton()).toBeDisabled();
    await user.type(input, "   ");
    expect(sendButton()).toBeDisabled();
    await user.keyboard("{Enter}");
    expect(onSend).not.toHaveBeenCalled();
  });

  it("sends the trimmed text on Enter and clears the input", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn().mockResolvedValue(undefined);
    renderWithProviders(
      <ReplyInput onSend={onSend} isSending={false} conversationId={1} />
    );
    const input = screen.getByPlaceholderText("Escribe un mensaje...");
    await user.type(input, "  Hola cliente  ");
    await user.keyboard("{Enter}");
    expect(onSend).toHaveBeenCalledTimes(1);
    expect(onSend).toHaveBeenCalledWith("Hola cliente");
    await expect.poll(() => (input as HTMLInputElement).value).toBe("");
  });

  it("does not send without a conversation selected", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn().mockResolvedValue(undefined);
    renderWithProviders(
      <ReplyInput onSend={onSend} isSending={false} conversationId={0} />
    );
    const input = screen.getByPlaceholderText("Selecciona una conversación para responder");
    expect(input).toBeDisabled();
    await user.type(input, "hola");
    expect(onSend).not.toHaveBeenCalled();
  });

  it("does not send while a send is in progress", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn().mockResolvedValue(undefined);
    renderWithProviders(
      <ReplyInput onSend={onSend} isSending={true} conversationId={1} value="hola" onChange={vi.fn()} />
    );
    await user.keyboard("{Enter}");
    expect(onSend).not.toHaveBeenCalled();
  });

  it("keeps the draft and shows an error toast when send fails", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn().mockRejectedValue(new Error("network"));
    renderWithProviders(
      <ReplyInput onSend={onSend} isSending={false} conversationId={1} />
    );
    const input = screen.getByPlaceholderText("Escribe un mensaje...");
    await user.type(input, "Hola cliente");
    await user.keyboard("{Enter}");
    expect(onSend).toHaveBeenCalledTimes(1);
    expect((input as HTMLInputElement).value).toBe("Hola cliente");
    expect(toast.error).toHaveBeenCalledWith(expect.stringMatching(/No se pudo enviar el mensaje/i));
  });

  it("is disabled without text or conversation and while in flight", () => {
    const onSend = vi.fn();
    const onChange = vi.fn();
    const { rerender } = renderWithProviders(
      <ReplyInput onSend={onSend} isSending={false} conversationId={1} />
    );
    // No text yet.
    expect(assistTrigger()).toBeDisabled();

    // Text but no conversation selected.
    rerender(<ReplyInput onSend={onSend} isSending={false} conversationId={0} value="hola" onChange={onChange} />);
    expect(assistTrigger()).toBeDisabled();

    // In-flight rephrase disables the actions.
    mocks.pending.value = true;
    rerender(<ReplyInput onSend={onSend} isSending={false} conversationId={1} value="hola" onChange={onChange} />);
    expect(assistTrigger()).toBeDisabled();
  });

  it("calls the rephrase mutation and replaces the input value without sending", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn().mockResolvedValue(undefined);
    mocks.rephrase.mockResolvedValue({ text: "Texto reformulado por la IA" });
    renderWithProviders(
      <ReplyInput onSend={onSend} isSending={false} conversationId={1} />
    );
    const input = screen.getByPlaceholderText("Escribe un mensaje...");
    await user.type(input, "hola");
    await user.click(assistTrigger());
    await user.click(screen.getByText(ui.agent.rephrase));
    expect(mocks.rephrase).toHaveBeenCalledWith({ text: "hola", mode: "rephrase" });
    await expect.poll(() => (input as HTMLInputElement).value).toBe("Texto reformulado por la IA");
    expect(onSend).not.toHaveBeenCalled();
  });

  it("keeps the draft and toasts on rephrase failure", async () => {
    const user = userEvent.setup();
    mocks.rephrase.mockRejectedValue(new Error("API Error 500: boom"));
    renderWithProviders(
      <ReplyInput onSend={vi.fn()} isSending={false} conversationId={1} />
    );
    const input = screen.getByPlaceholderText("Escribe un mensaje...");
    await user.type(input, "hola");
    await user.click(assistTrigger());
    await user.click(screen.getByText(ui.agent.rephrase));
    expect((input as HTMLInputElement).value).toBe("hola");
    expect(toast.error).toHaveBeenCalledWith(ui.agent.rephraseError);
  });

  it("shows the credits-exhausted toast on 402 and keeps the draft", async () => {
    const user = userEvent.setup();
    mocks.rephrase.mockRejectedValue(new Error("API Error 402: ai_credits_exhausted"));
    renderWithProviders(
      <ReplyInput onSend={vi.fn()} isSending={false} conversationId={1} />
    );
    const input = screen.getByPlaceholderText("Escribe un mensaje...");
    await user.type(input, "hola");
    await user.click(assistTrigger());
    await user.click(screen.getByText(ui.agent.rephrase));
    expect((input as HTMLInputElement).value).toBe("hola");
    expect(toast.error).toHaveBeenCalledWith(ui.agent.rephraseCreditsExhausted);
  });
});
