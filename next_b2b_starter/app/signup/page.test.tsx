import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";

import SignupPage from "./page";

vi.mock("@/lib/api/api/repositories/signup-repository", () => ({
  signupRepository: {
    createOrganizationWithMagicLink: vi.fn(),
    bootstrapOrganization: vi.fn(),
  },
}));

import { signupRepository } from "@/lib/api/api/repositories/signup-repository";

const mockCreate = vi.mocked(signupRepository.createOrganizationWithMagicLink);

async function completeAccountStep() {
  fireEvent.change(screen.getByPlaceholderText("Juan Pérez"), {
    target: { value: "Ana Gómez" },
  });
  fireEvent.change(screen.getByPlaceholderText("tu@empresa.com"), {
    target: { value: "ana@acme.com" },
  });
  fireEvent.click(screen.getByText("Continuar"));
}

async function completeOrganizationStep() {
  await screen.findByPlaceholderText("Acme S.A.S.");
  fireEvent.change(screen.getByPlaceholderText("Acme S.A.S."), {
    target: { value: "Acme SAS" },
  });
  fireEvent.click(screen.getByText("Continuar"));
}

describe("SignupPage wizard", () => {
  beforeEach(() => {
    localStorage.clear();
    mockCreate.mockReset();
    mockCreate.mockResolvedValue({
      orgId: "org_123",
      orgName: "Acme SAS",
      displayName: "Acme SAS",
      ownerUserId: "",
      ownerEmail: "ana@acme.com",
      ownerName: "Ana Gómez",
      loginUrl: "/auth",
    });
  });

  it("walks through account → organization → business and submits through magic link", async () => {
    render(<SignupPage />);

    await completeAccountStep();
    await completeOrganizationStep();

    await screen.findByText("Cuéntanos de tu negocio");
    fireEvent.change(screen.getByPlaceholderText("Ej: atender consultas de clientes, recibir pedidos, facturar…"), {
      target: { value: "Atender consultas de clientes por WhatsApp" },
    });
    fireEvent.click(screen.getByText("Crear cuenta"));

    await waitFor(() => expect(mockCreate).toHaveBeenCalledTimes(1));

    const [owner, organization] = mockCreate.mock.calls[0];
    expect(owner).toEqual({ fullName: "Ana Gómez", email: "ana@acme.com" });
    expect(organization).toEqual({ displayName: "Acme SAS", industry: "Technology" });
  });

  it("never passes owner_password through any wizard path", async () => {
    render(<SignupPage />);

    await completeAccountStep();
    await completeOrganizationStep();

    await screen.findByText("Cuéntanos de tu negocio");
    fireEvent.change(screen.getByPlaceholderText("Ej: atender consultas de clientes, recibir pedidos, facturar…"), {
      target: { value: "Recibir pedidos" },
    });
    fireEvent.click(screen.getByText("Crear cuenta"));

    await waitFor(() => expect(mockCreate).toHaveBeenCalledTimes(1));

    const [owner, organization] = mockCreate.mock.calls[0];
    expect(owner).not.toHaveProperty("owner_password");
    expect(owner).not.toHaveProperty("password");
    expect(organization).not.toHaveProperty("owner_password");
  });

  it("persists business context to localStorage on continue", async () => {
    render(<SignupPage />);

    await completeAccountStep();
    await completeOrganizationStep();

    await screen.findByText("Cuéntanos de tu negocio");
    fireEvent.click(screen.getByText("Sí, ya tengo WhatsApp Business"));
    fireEvent.change(screen.getByPlaceholderText("Ej: atender consultas de clientes, recibir pedidos, facturar…"), {
      target: { value: "Atención al cliente" },
    });
    fireEvent.click(screen.getByText("Crear cuenta"));

    await waitFor(() => expect(mockCreate).toHaveBeenCalledTimes(1));

    const stored = JSON.parse(
      localStorage.getItem("ai-onboarding.business-context") ?? "null"
    );
    expect(stored).toEqual({
      whatsappReadiness: "already",
      businessGoal: "Atención al cliente",
    });
  });
});
