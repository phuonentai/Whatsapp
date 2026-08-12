import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { UseMutationResult } from "@tanstack/react-query";
import { renderWithProviders } from "@/test/render";
import { CampaignManager } from "./campaign-manager";
import { AudienceResultCard } from "./audience-result-card";
import {
  useAiBuild,
  useCreateSegment,
} from "@/lib/hooks/mutations/use-campaign-mutations";
import type { AudienceBuildResultDto, SegmentDto, SegmentFilter } from "@/lib/api/api/dto/campaign.dto";

const AI_SPEC: SegmentFilter[] = [
  { field: "lead_status", op: "eq", value: "cliente" },
  { field: "recency_days", op: "lte", value: 30 },
];
const AI_MESSAGE = "¡Hola! Tenemos ofertas especiales para clientes mayoristas este mes. Responde SÍ para conocerlas.";
const AI_RESULT: AudienceBuildResultDto = {
  filter_spec: AI_SPEC,
  preview: { total: 120, excluded_by_gates: 8 },
  message_draft: AI_MESSAGE,
};

const createSegmentMutate = vi.fn();
const aiBuildMutate = vi.fn(
  (_text: string, opts: { onSuccess: (r: AudienceBuildResultDto) => void }) => {
    opts.onSuccess(AI_RESULT);
  }
);
const mockCreateCampaignMutate = vi.fn();
const mockLaunchCampaignMutate = vi.fn();
const mockSegments: SegmentDto[] = [
  {
    id: 1,
    organization_id: 42,
    nombre: "Clientes",
    filter_spec: AI_SPEC,
    created_at: "2026-08-11T00:00:00Z",
    updated_at: "2026-08-11T00:00:00Z",
  },
];

type CreateSegmentMutation = UseMutationResult<
  SegmentDto,
  Error,
  { nombre: string; filter_spec: SegmentFilter[] }
>;
type AiBuildMutation = UseMutationResult<AudienceBuildResultDto, Error, string>;

vi.mock("@/lib/hooks/queries/use-campaign-queries", () => ({
  useSegmentsQuery: () => ({ data: mockSegments, isLoading: false }),
  useCampaignsQuery: () => ({ data: [] }),
  useCampaignRecipientsQuery: vi.fn(() => ({ data: [], isLoading: false })),
}));

vi.mock("@/lib/hooks/mutations/use-campaign-mutations", () => ({
  useCreateSegment: vi.fn(),
  useDeleteSegment: () => ({ mutate: vi.fn(), isPending: false }),
  useCreateCampaign: () => ({ mutate: mockCreateCampaignMutate, isPending: false }),
  useLaunchCampaign: () => ({ mutate: mockLaunchCampaignMutate, isPending: false }),
  useAiBuild: vi.fn(),
}));

vi.mock("@/lib/api/api/repositories/campaign-repository", () => ({
  campaignRepository: {
    previewSpec: vi.fn().mockResolvedValue({ total: 5, excluded_by_gates: 0 }),
  },
}));

describe("AudienceResultCard", () => {
  it("renders the structured result without any raw JSON node", () => {
    const { container } = renderWithProviders(
      <AudienceResultCard
        spec={AI_SPEC}
        preview={AI_RESULT.preview}
        onAccept={vi.fn()}
        onEdit={vi.fn()}
        onRegenerate={vi.fn()}
        onPreview={vi.fn()}
      />
    );

    expect(screen.getByText("Audiencia generada")).toBeDefined();
    expect(screen.getByText("lead_status igual a cliente")).toBeDefined();
    expect(screen.getByText("recency_days menor o igual que 30")).toBeDefined();
    expect(screen.getByText("120")).toBeDefined();
    expect(screen.getByText(/8 excluidos por consentimiento/)).toBeDefined();
    expect(screen.getByText(/Ley 1581/)).toBeDefined();
    expect(screen.getByRole("button", { name: "Guardar como segmento" })).toBeDefined();
    expect(screen.getByRole("button", { name: "Editar descripción" })).toBeDefined();
    expect(screen.getByRole("button", { name: "Regenerar" })).toBeDefined();
    expect(screen.getByRole("button", { name: "Ver vista previa" })).toBeDefined();

    // No raw JSON in the card.
    expect(container.querySelector("pre")).toBeNull();
    expect(screen.queryByText(JSON.stringify(AI_SPEC, null, 2))).toBeNull();
  });
});

describe("CampaignManager", () => {
  beforeEach(() => {
    createSegmentMutate.mockReset();
    mockCreateCampaignMutate.mockReset();
    mockLaunchCampaignMutate.mockReset();
    vi.mocked(useCreateSegment).mockReturnValue({
      mutate: createSegmentMutate,
      isPending: false,
    } as unknown as CreateSegmentMutation);
    vi.mocked(useAiBuild).mockReturnValue({
      mutate: aiBuildMutate,
      isPending: false,
      isError: false,
      error: null,
    } as unknown as AiBuildMutation);
  });

  it("shows the audience card instead of a raw JSON dump and keeps the accept payload", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(<CampaignManager />);

    await user.type(
      screen.getByPlaceholderText("Ej: clientes mayoristas que escribieron este mes"),
      "clientes mayoristas de este mes"
    );
    await user.click(screen.getByRole("button", { name: "Generar audiencia" }));

    expect(screen.getByTestId("audience-result-card")).toBeDefined();
    expect(screen.getByText("lead_status igual a cliente")).toBeDefined();
    expect(screen.getByText("120")).toBeDefined();
    expect(screen.getByText(/8 excluidos por consentimiento/)).toBeDefined();

    // The AI result must not leak as raw JSON anywhere in the page.
    expect(screen.queryByText(JSON.stringify(AI_SPEC, null, 2))).toBeNull();
    const preNodes = container.querySelectorAll("pre");
    for (const pre of preNodes) {
      expect(pre.textContent).not.toContain("lead_status");
    }

    // Accept payload contract unchanged: same filter_spec as the AI result.
    await user.click(screen.getByRole("button", { name: "Guardar como segmento" }));
    expect(createSegmentMutate).toHaveBeenCalledWith(
      { nombre: "clientes mayoristas de este mes", filter_spec: AI_SPEC },
      expect.any(Object)
    );
  });

  it("pre-fills the message textarea with the AI message_draft after aiBuild", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CampaignManager />);

    await user.type(
      screen.getByPlaceholderText("Ej: clientes mayoristas que escribieron este mes"),
      "clientes mayoristas de este mes"
    );
    await user.click(screen.getByRole("button", { name: "Generar audiencia" }));

    const textarea = screen.getByLabelText("Mensaje de la campaña (opcional)") as HTMLTextAreaElement;
    expect(textarea.value).toBe(AI_MESSAGE);
    expect(screen.getByText(/La IA redactó un borrador de mensaje/)).toBeDefined();
  });

  it("keeps user edits to the draft and sends them in the create payload", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CampaignManager />);

    await user.type(
      screen.getByPlaceholderText("Ej: clientes mayoristas que escribieron este mes"),
      "clientes mayoristas"
    );
    await user.click(screen.getByRole("button", { name: "Generar audiencia" }));

    const textarea = screen.getByLabelText("Mensaje de la campaña (opcional)");
    await user.clear(textarea);
    await user.type(textarea, "Oferta editada por el usuario: 20% dcto. esta semana.");

    await user.type(screen.getByPlaceholderText("Nombre de la campaña"), "Promo clientes");
    await user.selectOptions(screen.getByRole("combobox"), "1");
    await user.click(screen.getByRole("button", { name: "Crear campaña" }));

    expect(mockCreateCampaignMutate).toHaveBeenCalledWith(
      {
        nombre: "Promo clientes",
        segment_id: 1,
        mensaje: "Oferta editada por el usuario: 20% dcto. esta semana.",
      },
      expect.any(Object)
    );
  });

  it("creates a campaign without mensaje when the message is empty", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CampaignManager />);

    await user.type(screen.getByPlaceholderText("Nombre de la campaña"), "Promo sin mensaje");
    await user.selectOptions(screen.getByRole("combobox"), "1");
    await user.click(screen.getByRole("button", { name: "Crear campaña" }));

    expect(mockCreateCampaignMutate).toHaveBeenCalledWith(
      { nombre: "Promo sin mensaje", segment_id: 1 },
      expect.any(Object)
    );
    const payload = mockCreateCampaignMutate.mock.calls[0][0] as Record<string, unknown>;
    expect("mensaje" in payload).toBe(false);
  });

  it("does not launch or send anything when a draft is created", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CampaignManager />);

    await user.type(
      screen.getByPlaceholderText("Ej: clientes mayoristas que escribieron este mes"),
      "clientes mayoristas"
    );
    await user.click(screen.getByRole("button", { name: "Generar audiencia" }));
    await user.type(screen.getByPlaceholderText("Nombre de la campaña"), "Promo clientes");
    await user.selectOptions(screen.getByRole("combobox"), "1");
    await user.click(screen.getByRole("button", { name: "Crear campaña" }));

    expect(mockCreateCampaignMutate).toHaveBeenCalled();
    expect(mockLaunchCampaignMutate).not.toHaveBeenCalled();
  });
});
