import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { renderWithProviders } from "@/test/render";
import { ReplyInput } from "./reply-input";

describe("ReplyInput", () => {
  it("disables send for empty/whitespace input", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn().mockResolvedValue(undefined);
    renderWithProviders(
      <ReplyInput onSend={onSend} isSending={false} conversationId={1} />
    );
    const input = screen.getByPlaceholderText("Type a message...");
    const send = screen.getByRole("button");
    expect(send).toBeDisabled();
    await user.type(input, "   ");
    expect(send).toBeDisabled();
    await user.keyboard("{Enter}");
    expect(onSend).not.toHaveBeenCalled();
  });

  it("sends the trimmed text on Enter and clears the input", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn().mockResolvedValue(undefined);
    renderWithProviders(
      <ReplyInput onSend={onSend} isSending={false} conversationId={1} />
    );
    const input = screen.getByPlaceholderText("Type a message...");
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
    const input = screen.getByPlaceholderText("Select a conversation to reply");
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
    const input = screen.getByPlaceholderText("Type a message...");
    await user.type(input, "Hola cliente");
    await user.keyboard("{Enter}");
    expect(onSend).toHaveBeenCalledTimes(1);
    expect((input as HTMLInputElement).value).toBe("Hola cliente");
    expect(toast.error).toHaveBeenCalledWith(expect.stringMatching(/No se pudo enviar el mensaje/i));
  });
});
