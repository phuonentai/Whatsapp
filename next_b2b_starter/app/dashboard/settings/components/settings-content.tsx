"use client";

import { useEffect, useMemo, useState } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import {
  User,
  Users,
  CreditCard,
  RefreshCcw,
  ArrowLeft,
  ChevronRight,
  Boxes,
  Bot,
  ShieldCheck,
  MessageCircle,
  Instagram,
  ScrollText,
  FileText,
  ServerCog,
  MessageSquarePlus,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { format } from "date-fns";

import { ProfileSection } from "./profile-section";
import { MemberList } from "./member-list";
import { InviteMember } from "./invite-member";
import { MemberHelpers } from "@/lib/models/member.model";
import { usePermissions } from "@/lib/hooks/use-permissions";
import { PERMISSIONS } from "@/lib/auth/permissions";
import { useWhatsAppConfigQuery } from "@/lib/hooks/queries/use-whatsapp-config-query";
import { WhatsAppConfigSection } from "./whatsapp-config-section";
import { TemplatesSection } from "./templates-section";
import { useInstagramConfigQuery } from "@/lib/hooks/queries/use-instagram-config-query";
import { InstagramConfigSection } from "./instagram-config-section";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { SubscriptionGateState } from "@/lib/polar/current-subscription";
import { SubscriptionTab } from "./subscription-tab";
import { ModulesSection } from "./modules-section";
import { PlaybookSetupCard } from "./playbooks-section";
import { AgentSettingsSection } from "./agent-settings-section";
import { ComplianceSection } from "./compliance-section";
import { SecuritySection } from "./security-section";
import { MfaPolicySection } from "./mfa-policy-section";
import { AuditLogView } from "./audit-log-view";
import { SiigoIntegrationSection } from "./siigo-integration-section";
import { SiigoAdminView } from "./siigo-admin-view";
import { EquipoPermisos } from "./equipo-permisos";
import { SsoAdminView } from "./sso-admin-view";
import { ScimAdminView } from "./scim-admin-view";

// Query hooks - Component depends ONLY on these hooks
import { useProfileQuery } from "@/lib/hooks/queries/use-profile-query";
import { useMembersQuery } from "@/lib/hooks/queries/use-members-query";
import { useSubscriptionQuery } from "@/lib/hooks/queries/use-subscription-query";
import { useInviteMember } from "@/lib/hooks/mutations/use-invite-member";
import type { InviteMemberRequest } from "@/lib/models/member.model";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { ui } from "@/lib/copy/ui";

interface SettingsContentProps {
  // No props required - component fetches its own data
}

type SettingsView = "overview" | "profile" | "members" | "subscription" | "modules" | "ai" | "compliance" | "audit" | "whatsapp" | "instagram" | "siigo" | "siigo-admin" | "templates" | "access" | "sso" | "scim";

interface OverviewSection {
  key: Exclude<SettingsView, "overview">;
  title: string;
  description: string;
  value: string;
  helper: string;
  icon: LucideIcon;
  disabled?: boolean;
}

const DETAIL_META: Record<Exclude<SettingsView, "overview">, { title: string; description: string }> = {
  profile: {
    title: "Account & workspace",
    description: "Update your profile details and workspace metadata.",
  },
  members: {
    title: "Team access",
    description: "Invite new teammates, manage existing members, and adjust permissions.",
  },
  subscription: {
    title: "Subscription & billing",
    description: "Review your subscription status, usage limits, and cancellation controls.",
  },
  modules: {
    title: "Modules",
    description: "Enable product modules and adjust each one to your needs.",
  },
  ai: {
    title: "AI Copilot",
    description: "Configure the WhatsApp AI assistant: mode, tone, guardrails, and consent.",
  },
  compliance: {
    title: "Compliance (Ley 1581)",
    description: "Data consent, export, and right-to-erasure controls for your contacts.",
  },
  audit: {
    title: "Audit log",
    description: "Read-only record of activity across your organization.",
  },
  whatsapp: {
    title: "Messaging",
    description: "Connect and manage your WhatsApp Business integration.",
  },
  templates: {
    title: "Message templates",
    description: "Manage Meta-approved WhatsApp message templates.",
  },
  instagram: {
    title: "Instagram",
    description: "Connect and manage your Instagram DMs integration.",
  },
  siigo: {
    title: "Integración Siigo",
    description: "Conecta Siigo para facturación electrónica automática.",
  },
  "siigo-admin": {
    title: "Onboarding Siigo",
    description: "Vista de operación: estado de conexión por organización.",
  },
  access: {
    title: "Equipo y permisos",
    description: "Miembros, matriz de permisos y módulos activos del espacio de trabajo.",
  },
  sso: {
    title: "SSO (SAML / OIDC)",
    description: "Conexiones de inicio de sesión único gestionadas en el portal de Stytch.",
  },
  scim: {
    title: "SCIM",
    description: "Aprovisionamiento de usuarios y roles desde el directorio de la empresa.",
  },
};

// Design language: tinted icon tile per module (identity: icons on `-50`
// tinted tiles with matching accent). Used by the overview list and the
// detail-page summary card.
const TILE_TONES: Record<
  Exclude<SettingsView, "overview">,
  { tile: string; icon: string }
> = {
  profile: { tile: "bg-blue-50", icon: "text-blue-600" },
  members: { tile: "bg-violet-50", icon: "text-violet-600" },
  subscription: { tile: "bg-cyan-50", icon: "text-cyan-600" },
  modules: { tile: "bg-indigo-50", icon: "text-indigo-600" },
  ai: { tile: "bg-purple-50", icon: "text-purple-600" },
  compliance: { tile: "bg-teal-50", icon: "text-teal-600" },
  audit: { tile: "bg-slate-100", icon: "text-slate-600" },
  whatsapp: { tile: "bg-emerald-50", icon: "text-emerald-600" },
  templates: { tile: "bg-sky-50", icon: "text-sky-600" },
  instagram: { tile: "bg-pink-50", icon: "text-pink-600" },
  siigo: { tile: "bg-orange-50", icon: "text-orange-600" },
  "siigo-admin": { tile: "bg-amber-50", icon: "text-amber-600" },
  access: { tile: "bg-violet-50", icon: "text-violet-600" },
  sso: { tile: "bg-blue-50", icon: "text-blue-600" },
  scim: { tile: "bg-cyan-50", icon: "text-cyan-600" },
};

function parseViewParam(raw: string | null): SettingsView | null {
  if (!raw) return null;
  const normalized = raw.toLowerCase();
  if (
    normalized === "profile" ||
    normalized === "members" ||
    normalized === "subscription" ||
    normalized === "modules" ||
    normalized === "ai" ||
    normalized === "compliance" ||
    normalized === "audit" ||
    normalized === "whatsapp" ||
    normalized === "templates" ||
    normalized === "instagram" ||
    normalized === "siigo" ||
    normalized === "siigo-admin" ||
    normalized === "access" ||
    normalized === "sso" ||
    normalized === "scim"
  ) {
    return normalized as SettingsView;
  }
  return null;
}

function getPlanNameFromRecord(record: Record<string, unknown> | null | undefined) {
  if (!record || typeof record !== "object") {
    return null;
  }

  const planKeys = [
    "plan_name",
    "plan_label",
    "plan_display_name",
    "subscription_name",
    "product_name",
    "name",
  ];

  for (const key of planKeys) {
    const value = record[key];
    if (typeof value === "string") {
      const trimmed = value.trim();
      if (trimmed.length > 0) {
        return trimmed;
      }
    }
  }

  return null;
}

function resolvePlanLabel(state: SubscriptionGateState | null): string {
  if (!state) {
    return "Active plan";
  }

  const planNameFromSubscription = getPlanNameFromRecord(
    state.subscription?.metadata ?? undefined
  );
  if (planNameFromSubscription) {
    return planNameFromSubscription;
  }

  const planNameFromCustomFields = getPlanNameFromRecord(
    state.subscription?.customFieldData ?? undefined
  );
  if (planNameFromCustomFields) {
    return planNameFromCustomFields;
  }

  const planNameFromProduct = getPlanNameFromRecord(
    state.subscription?.productMetadata ?? undefined
  );
  if (planNameFromProduct) {
    return planNameFromProduct;
  }

  if (state.subscription?.productName) {
    return state.subscription.productName;
  }

  return "Active plan";
}

function getSubscriptionQuickStatus(
  state: SubscriptionGateState | null,
  isLoading: boolean
) {
  if (isLoading && !state) {
    return {
      title: "Loading…",
      helper: "Fetching your Polar subscription.",
    };
  }

  if (!state) {
    return {
      title: "No active plan",
      helper: "Choose a plan below to unlock automations.",
    };
  }

  if (state.reason === "POLAR_UNCONFIGURED") {
    return {
      title: "Setup required",
      helper: "Add Polar credentials in the environment to enable billing.",
    };
  }

  if (state.reason === "BACKEND_UNAVAILABLE") {
    return {
      title: "Temporarily unavailable",
      helper: "We're still connecting to Polar. Try refreshing shortly.",
    };
  }

  if (!state.isActive || state.reason === "NO_ACTIVE_SUBSCRIPTION") {
    return {
      title: "No active plan",
      helper: "Select a plan below to keep automations running.",
    };
  }

  if (state.subscription?.cancelAtPeriodEnd) {
    const cancellationDate = state.subscription.currentPeriodEnd
      ? format(new Date(state.subscription.currentPeriodEnd), "MMM d, yyyy")
      : null;

    return {
      title: "Cancels soon",
      helper: cancellationDate
        ? `Ends on ${cancellationDate}. Update your plan below to stay active.`
        : "Scheduled to cancel at period end.",
      };
  }

  const planLabel = resolvePlanLabel(state);
  const renewalDate = state.subscription?.currentPeriodEnd
    ? format(new Date(state.subscription.currentPeriodEnd), "MMM d, yyyy")
    : null;

  return {
    title: planLabel,
    helper: renewalDate ? `Renews on ${renewalDate}.` : "Billing is managed through Polar.",
  };
}

export function SettingsContent({}: SettingsContentProps = {}) {
  const {
    hasPermission,
    isInitialized: permissionsReady,
  } = usePermissions();
  const canManageMembers = hasPermission(PERMISSIONS.ORG_MANAGE);
  const canViewAudit = hasPermission(PERMISSIONS.AUDIT_VIEW);
  const hasSubscriptionPermission = hasPermission(PERMISSIONS.ORG_MANAGE);
  const shouldLoadSubscription = permissionsReady && hasSubscriptionPermission;

  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const viewParam = searchParams.get("view");

  const [viewStack, setViewStack] = useState<SettingsView[]>(["overview"]);
  const currentView = viewStack[viewStack.length - 1];
  const [isInviteModalOpen, setInviteModalOpen] = useState(false);

  // Render-phase sync of the settings view stack with the ?view= URL param.
  // Guards prevent infinite loops; setState here is the React-sanctioned
  // "adjust state during render" pattern (avoids setState-in-effect).
  const [prevRequested, setPrevRequested] = useState<SettingsView | null>(null);
  const requestedView = permissionsReady ? parseViewParam(viewParam) : null;
  if (requestedView !== prevRequested) {
    setPrevRequested(requestedView);
    if (requestedView) {
      const isAllowed =
        (requestedView === "members" && canManageMembers) ||
        (requestedView === "access" && canManageMembers) ||
        (requestedView === "subscription" && hasSubscriptionPermission) ||
        (requestedView === "ai" && canManageMembers) ||
        (requestedView === "compliance" && canManageMembers) ||
        (requestedView === "audit" && canViewAudit) ||
        (requestedView === "whatsapp" && canManageMembers) ||
        (requestedView === "templates" && canManageMembers) ||
        (requestedView === "instagram" && canManageMembers) ||
        (requestedView === "siigo" && canManageMembers) ||
        (requestedView === "siigo-admin" && canManageMembers) ||
        (requestedView === "sso" && canManageMembers) ||
        (requestedView === "scim" && canManageMembers);
      if (isAllowed) {
        setViewStack((stack) =>
          stack[stack.length - 1] === requestedView ? stack : ["overview", requestedView]
        );
      }
    }
  }

  if (
    currentView === "subscription" &&
    !hasSubscriptionPermission &&
    viewStack[viewStack.length - 1] !== "overview"
  ) {
    setViewStack(["overview"]);
  }
  if (
    (currentView === "members" ||
      currentView === "access" ||
      currentView === "ai" ||
      currentView === "compliance" ||
      currentView === "whatsapp" ||
      currentView === "templates" ||
      currentView === "instagram" ||
      currentView === "sso" ||
      currentView === "scim") &&
    !canManageMembers &&
    viewStack[viewStack.length - 1] !== "overview"
  ) {
    setViewStack(["overview"]);
  }
  if (currentView === "audit" && !canViewAudit && viewStack[viewStack.length - 1] !== "overview") {
    setViewStack(["overview"]);
  }

  useEffect(() => {
    if (!permissionsReady) return;
    const requested = parseViewParam(viewParam);
    if (!requested) return;
    if (requested === "members" && !canManageMembers) return;
    if (requested === "access" && !canManageMembers) return;
    if (requested === "subscription" && !hasSubscriptionPermission) return;
    if (requested === "ai" && !canManageMembers) return;
    if (requested === "compliance" && !canManageMembers) return;
    if (requested === "audit" && !canViewAudit) return;
    if (requested === "whatsapp" && !canManageMembers) return;
    if (requested === "templates" && !canManageMembers) return;
    if (requested === "instagram" && !canManageMembers) return;
    if (requested === "sso" && !canManageMembers) return;
    if (requested === "scim" && !canManageMembers) return;

    // Intentionally syncs the view stack from the URL query param (deep links,
    // refresh, back/forward). Cannot be derived during render without losing the
    // in-app back-stack semantics.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setViewStack((stack) => {
      if (stack[stack.length - 1] === requested) {
        return stack;
      }
      return ["overview", requested];
    });
  }, [permissionsReady, canManageMembers, hasSubscriptionPermission, canViewAudit, viewParam]);

  const pushView = (view: Exclude<SettingsView, "overview">) => {
    setViewStack((stack) => {
      if (stack[stack.length - 1] === view) {
        return stack;
      }
      return [...stack, view];
    });
  };

  const goBack = () => {
    setViewStack((stack) => (stack.length > 1 ? stack.slice(0, -1) : stack));
  };

  useEffect(() => {
    const desiredParam = currentView === "overview" ? null : currentView;
    if (desiredParam === viewParam) {
      return;
    }
    const params = new URLSearchParams(searchParams.toString());
    if (desiredParam) {
      params.set("view", desiredParam);
    } else {
      params.delete("view");
    }
    const queryString = params.toString();
    router.replace(queryString ? `${pathname}?${queryString}` : pathname, { scroll: false });
  }, [currentView, viewParam, router, pathname, searchParams]);

  // Use query hooks - data is cached and reused globally
  const {
    data: profile,
    isLoading: isProfileLoading,
    error: profileError,
    refetch: refetchProfile,
  } = useProfileQuery({
    enabled: permissionsReady,
  });

  const {
    data: membersData,
    isLoading: isMembersLoading,
    isFetching: isMembersFetching,
    error: membersError,
    refetch: refetchMembers,
  } = useMembersQuery(
    {
      organizationId: profile?.organizationId,
      page: 1,
      pageSize: 50,
      enabled: canManageMembers && Boolean(profile?.organizationId),
    }
  );

  const {
    data: subscriptionState,
    isLoading: isSubscriptionLoading,
    error: subscriptionError,
    refetch: refetchSubscription,
  } = useSubscriptionQuery({
    enabled: shouldLoadSubscription,
  });

  const { data: whatsappConfig } = useWhatsAppConfigQuery({
    enabled: permissionsReady && canManageMembers,
  });

  const { data: instagramConfig } = useInstagramConfigQuery({
    enabled: permissionsReady && canManageMembers,
  });

  // Mutations
  const inviteMemberMutation = useInviteMember();

  // Memoized values - MUST be before any early returns (React Hooks rules)
  const members = membersData?.members ?? [];
  const organizationId = profile?.organizationId ?? "";
  const hasOrganization = Boolean(organizationId);
  const canInviteMembers = canManageMembers && hasOrganization;
  const canViewMembers = canManageMembers && hasOrganization;

  const subscriptionQuick = useMemo(() => {
    if (!hasSubscriptionPermission) {
      return null;
    }

    return getSubscriptionQuickStatus(
      subscriptionState ?? null,
      isSubscriptionLoading
    );
  }, [hasSubscriptionPermission, subscriptionState, isSubscriptionLoading]);

  const roleConfig = useMemo(
    () => profile ? MemberHelpers.getRoleConfig(profile.role) : MemberHelpers.getRoleConfig("member"),
    [profile]
  );

  const membersErrorMessage = membersError?.message ?? null;
  const subscriptionErrorMessage = subscriptionError?.message ?? null;

  const overviewSections = useMemo<OverviewSection[]>(() => {
    if (!profile) {
      return [];
    }

    const sections: OverviewSection[] = [];

    const accountLabel =
      profile.name?.trim().length
        ? profile.name
        : profile.email ?? "Account";
    const workspaceLabel =
      profile.organizationName?.trim().length
        ? profile.organizationName
        : "No workspace assigned";

    sections.push({
      key: "profile",
      title: "Account & workspace",
      description: "Profile identity, workspace label, and contact details.",
      value: accountLabel,
      helper: `${workspaceLabel} • ${roleConfig.label}`,
      icon: User,
    });

    if (canManageMembers) {
      const disabled = !hasOrganization;

      let value = "Invite teammates";
      let helper = "Bring collaborators into the workflow.";

      if (disabled) {
        value = "No organization";
        helper = "Join or create an organization to manage team access.";
      } else if (membersErrorMessage) {
        value = "Needs attention";
        helper = membersErrorMessage;
      } else if (isMembersLoading && members.length === 0) {
        value = "Loading…";
        helper = "Fetching your team roster.";
      } else if (members.length > 0) {
        value = `${members.length} ${members.length === 1 ? "member" : "members"}`;
        helper = "Manage roles, invitations, and permissions.";
      }

      sections.push({
        key: "members",
        title: "Team access",
        description: "Invite teammates and fine-tune their permissions.",
        value,
        helper,
        icon: Users,
        disabled,
      });

      // Vista consolidada "Equipo y permisos" (view=access): overview section
      // with a summary (member count + own role); click navigates to
      // `?view=access`. Gate: org:manage (allowlist above).
      sections.push({
        key: "access",
        title: ui.teamPermissions.title,
        description: ui.teamPermissions.description,
        value: disabled
          ? "No organization"
          : membersErrorMessage
            ? "Needs attention"
            : isMembersLoading && members.length === 0
              ? "Loading…"
              : members.length > 0
                ? `${members.length} ${members.length === 1 ? "miembro" : "miembros"}`
                : "Invitar equipo",
        helper: disabled
          ? "Join or create an organization to manage access."
          : `${roleConfig.label} · ${ui.teamPermissions.policySource}`,
        icon: Users,
        disabled,
      });

      // SSO / SCIM enterprise surfaces (stytch-enterprise-suite): Admin
      // Portal views, org:manage gated. Entries render only for admins.
      sections.push({
        key: "sso",
        title: "SSO (SAML / OIDC)",
        description: "Inicio de sesión único con el IdP de tu empresa.",
        value: "Gestionar conexiones",
        helper: "Conexiones y credenciales gestionadas en el portal de Stytch.",
        icon: ShieldCheck,
      });
      sections.push({
        key: "scim",
        title: "SCIM",
        description: "Aprovisionamiento automático desde el directorio.",
        value: "Gestionar conexiones",
        helper: "Sincroniza usuarios y roles desde Okta, Azure AD, etc.",
        icon: ServerCog,
      });
    }

    if (hasSubscriptionPermission) {
      let value = "Open details";
      let helper = "Review plans, renewals, usage, and invoices.";

      if (subscriptionErrorMessage) {
        value = "Needs attention";
        helper = subscriptionErrorMessage;
      } else if (subscriptionQuick) {
        value = subscriptionQuick.title;
        helper = subscriptionQuick.helper;
      } else if (isSubscriptionLoading) {
        value = "Loading…";
        helper = "Fetching your Polar subscription.";
      }

      sections.push({
        key: "subscription",
        title: "Subscription & billing",
        description: "Manage plan changes, billing history, and usage.",
        value,
        helper,
        icon: CreditCard,
      });
    }

    sections.push({
      key: "modules",
      title: "Modules",
      description: "Enable product modules and configure them for your workspace.",
      value: "Manage modules",
      helper: "Tickets, agents, analytics and more.",
      icon: Boxes,
    });

    if (canManageMembers) {
      sections.push({
        key: "ai",
        title: "AI Copilot",
        description: "WhatsApp AI assistant: mode, tone, guardrails, consent.",
        value: "Configure assistant",
        helper: "Autopilot window, escalation rules and brand voice.",
        icon: Bot,
      });

      sections.push({
        key: "compliance",
        title: "Compliance (Ley 1581)",
        description: "Consent, data export, and right-to-erasure controls.",
        value: "Manage data rights",
        helper: "Habeas Data export and anonymization.",
        icon: ShieldCheck,
      });

      const whatsAppValue = whatsappConfig?.businessPhone || "Not connected";
      const whatsAppHelper = whatsappConfig ? "Active — manage your connection." : "Connect WhatsApp to receive messages";
      sections.push({
        key: "whatsapp",
        title: "Messaging",
        description: "Connect and manage your WhatsApp Business integration.",
        value: whatsAppValue,
        helper: whatsAppHelper,
        icon: MessageCircle,
      });

      const instagramValue = instagramConfig?.igUsername || "Not connected";
      const instagramHelper = instagramConfig
        ? `${instagramConfig.isActive ? "Active" : "Paused"} — manage your connection.`
        : "Connect Instagram to receive DMs";
      sections.push({
        key: "instagram",
        title: "Instagram",
        description: "Connect and manage your Instagram DMs integration.",
        value: instagramValue,
        helper: instagramHelper,
        icon: Instagram,
      });

      sections.push({
        key: "siigo",
        title: "Integración Siigo",
        description: "Facturación electrónica automática desde WhatsApp.",
        value: "Configurar facturación",
        helper: "Conecta Siigo para emitir facturas en la etapa facturado.",
        icon: FileText,
      });

      sections.push({
        key: "templates",
        title: ui.templates.title,
        description: ui.templates.description,
        value: "Administrar plantillas",
        helper: "Plantillas aprobadas por Meta para enviar fuera de la ventana de 24 h.",
        icon: MessageSquarePlus,
      });

      sections.push({
        key: "siigo-admin",
        title: "Onboarding Siigo",
        description: "Vista de operación: estado de conexión por organización.",
        value: "Ver organizaciones",
        helper: "Estado, numeración e importación de cada cliente.",
        icon: ServerCog,
      });
    }

    if (canViewAudit) {
      sections.push({
        key: "audit",
        title: "Audit log",
        description: "Read-only record of activity across the organization.",
        value: "View activity",
        helper: "Notes, calls, emails, tasks, and system events.",
        icon: ScrollText,
      });
    }

    return sections;
  }, [
    profile,
    roleConfig.label,
    canManageMembers,
    hasOrganization,
    members.length,
    membersErrorMessage,
    isMembersLoading,
    hasSubscriptionPermission,
    subscriptionQuick,
    isSubscriptionLoading,
    subscriptionErrorMessage,
    canViewAudit,
    whatsappConfig,
    instagramConfig,
  ]);

  // Handle invite member
  const handleInvite = async (request: InviteMemberRequest) => {
    if (!profile?.organizationId) {
      return;
    }

    try {
      await inviteMemberMutation.mutateAsync({
        request,
        organizationId: profile.organizationId,
      });
      setInviteModalOpen(false);
    } catch {
      // error toast handled by mutation
    }
    // Members list automatically refetches due to invalidation in mutation
  };

  // Handle refresh members (manual refetch)
  const handleRefreshMembers = () => {
    refetchMembers();
  };

  // Handle refresh subscription
  const handleRefreshSubscription = async () => {
    await refetchSubscription();
  };

  const isOverview = currentView === "overview";
  const activeDetailMeta =
    currentView === "overview"
      ? null
      : DETAIL_META[currentView as Exclude<SettingsView, "overview">];
  const activeSectionSummary = overviewSections.find(
    (section) => section.key === currentView
  );

  const renderDetailContent = () => {
    if (!profile) {
      return null;
    }

    switch (currentView) {
      case "profile":
        return (
          <div className="space-y-6">
            <ProfileSection profile={profile} />
            <SecuritySection />
          </div>
        );
      case "members":
        if (!canManageMembers || !hasOrganization) {
          return (
            <Alert className="border border-amber-200 bg-amber-50">
              <AlertTitle>Organization required</AlertTitle>
              <AlertDescription>
                Join or create an organization to manage team members.
              </AlertDescription>
            </Alert>
          );
        }
        return (
          <>
            <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="space-y-1">
                <h3 className="text-xl font-semibold text-slate-900">Team roster</h3>
                <p className="text-sm text-slate-600">
                  Review every teammate in your workspace and keep roles current.
                </p>
              </div>
              <Button
                onClick={() => setInviteModalOpen(true)}
                disabled={!canInviteMembers}
                className="w-full bg-emerald-500 text-white hover:bg-emerald-600 sm:w-auto"
              >
                Add member
              </Button>
            </div>

            {membersError && (
              <Alert
                variant="destructive"
                className="border border-red-200 bg-red-50"
              >
                <AlertTitle>Error</AlertTitle>
                <AlertDescription>
                  {membersError.message || "Failed to load team members"}
                </AlertDescription>
              </Alert>
            )}

            {isMembersFetching && members.length > 0 && (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <span className="inline-flex h-3 w-3 animate-spin rounded-full border-2 border-gray-200 border-t-primary-500" />
                Refreshing team roster…
              </div>
            )}

            {canViewMembers ? (
              <MemberList
                members={members}
                canManage={canManageMembers}
                currentUserId={profile.id}
                organizationId={organizationId}
                isFetching={isMembersFetching}
                onMemberUpdate={handleRefreshMembers}
              />
            ) : (
              <p className="text-sm text-muted-foreground">
                Join or switch into an organization to manage members.
              </p>
            )}

            <Dialog
              open={isInviteModalOpen}
              onOpenChange={(open) => {
                if (inviteMemberMutation.isPending) {
                  return;
                }
                setInviteModalOpen(open);
              }}
            >
              <DialogContent id="invite-member-dialog" className="sm:max-w-lg">
                <DialogHeader className="space-y-2 text-left">
                  <DialogTitle className="text-xl font-semibold text-slate-900">
                    Add a teammate
                  </DialogTitle>
                  <DialogDescription className="text-sm text-slate-600">
                    Send a secure invitation and assign the right access before they join.
                  </DialogDescription>
                </DialogHeader>
                <div className="pt-4">
                  <InviteMember
                    canInvite={canInviteMembers}
                    onInvite={handleInvite}
                  />
                </div>
              </DialogContent>
            </Dialog>
          </>
        );
      case "access":
        // 403 sin datos: el allowlist de gates ya bloquea la vista; el
        // componente aplica su propio gate de defensa en profundidad.
        if (!canManageMembers) {
          return (
            <Alert variant="destructive" className="border border-red-200 bg-red-50">
              <AlertTitle>Acceso restringido</AlertTitle>
              <AlertDescription>
                No tienes permisos para gestionar el equipo y los permisos.
              </AlertDescription>
            </Alert>
          );
        }
        return <EquipoPermisos />;
      case "sso":
        // Admin Portal SSO surface (enterprise-sso capability). Gated by
        // org:manage via the view allowlist above + the component's own gate.
        if (!canManageMembers) {
          return (
            <Alert variant="destructive" className="border border-red-200 bg-red-50">
              <AlertTitle>Acceso restringido</AlertTitle>
              <AlertDescription>
                No tienes permisos para administrar las conexiones SSO.
              </AlertDescription>
            </Alert>
          );
        }
        return <SsoAdminView />;
      case "scim":
        // Admin Portal SCIM surface (scim-provisioning capability). Gated by
        // org:manage via the view allowlist above + the component's own gate.
        if (!canManageMembers) {
          return (
            <Alert variant="destructive" className="border border-red-200 bg-red-50">
              <AlertTitle>Acceso restringido</AlertTitle>
              <AlertDescription>
                No tienes permisos para administrar el aprovisionamiento SCIM.
              </AlertDescription>
            </Alert>
          );
        }
        return <ScimAdminView />;
      case "subscription":
        if (!hasSubscriptionPermission) {
          return (
            <Alert variant="destructive" className="border border-red-200 bg-red-50">
              <AlertTitle>Access restricted</AlertTitle>
              <AlertDescription>
                You don&apos;t have permission to manage subscription or billing settings.
              </AlertDescription>
            </Alert>
          );
        }

        return (
          <SubscriptionTab
            state={shouldLoadSubscription ? subscriptionState ?? null : null}
            isLoading={shouldLoadSubscription ? isSubscriptionLoading : false}
            error={shouldLoadSubscription ? subscriptionErrorMessage : null}
            onRefresh={handleRefreshSubscription}
          />
        );
      case "modules":
        return <ModulesSection />;
      case "ai":
        return <AgentSettingsSection />;
      case "compliance":
        return (
          <div className="space-y-6">
            <ComplianceSection />
            <MfaPolicySection profile={profile} />
          </div>
        );
      case "audit":
        if (!canViewAudit) {
          return (
            <Alert variant="destructive" className="border border-red-200 bg-red-50">
              <AlertTitle>Access restricted</AlertTitle>
              <AlertDescription>
                You don&apos;t have permission to view the audit log.
              </AlertDescription>
            </Alert>
          );
        }
        return <AuditLogView />;
      case "whatsapp":
        if (!canManageMembers) {
          return (
            <Alert variant="destructive" className="border border-red-200 bg-red-50">
              <AlertTitle>Access restricted</AlertTitle>
              <AlertDescription>
                You don&apos;t have permission to manage the WhatsApp integration.
              </AlertDescription>
            </Alert>
          );
        }
        return <WhatsAppConfigSection />;
      case "instagram":
        if (!canManageMembers) {
          return (
            <Alert variant="destructive" className="border border-red-200 bg-red-50">
              <AlertTitle>Access restricted</AlertTitle>
              <AlertDescription>
                You don&apos;t have permission to manage the Instagram integration.
              </AlertDescription>
            </Alert>
          );
        }
        return <InstagramConfigSection />;
      case "templates":
        if (!canManageMembers) {
          return (
            <Alert variant="destructive" className="border border-red-200 bg-red-50">
              <AlertTitle>Access restricted</AlertTitle>
              <AlertDescription>
                You don&apos;t have permission to manage message templates.
              </AlertDescription>
            </Alert>
          );
        }
        return <TemplatesSection />;
      case "siigo":
        if (!canManageMembers) {
          return (
            <Alert variant="destructive" className="border border-red-200 bg-red-50">
              <AlertTitle>Acceso restringido</AlertTitle>
              <AlertDescription>
                No tienes permisos para administrar la integración Siigo.
              </AlertDescription>
            </Alert>
          );
        }
        return <SiigoIntegrationSection />;
      case "siigo-admin":
        if (!canManageMembers) {
          return (
            <Alert variant="destructive" className="border border-red-200 bg-red-50">
              <AlertTitle>Acceso restringido</AlertTitle>
              <AlertDescription>
                Solo los administradores pueden ver el onboarding Siigo.
              </AlertDescription>
            </Alert>
          );
        }
        return <SiigoAdminView />;
      default:
        return null;
    }
  };

  // Loading state
  const isLoading = isProfileLoading || !permissionsReady;

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-[600px] rounded-xl" />
      </div>
    );
  }

  // Error state
  if (profileError || !profile) {
    return (
      <div className="flex min-h-[400px] flex-col items-center justify-center space-y-4 rounded-xl border border-red-200 bg-red-50 px-6 py-12 text-center">
        <p className="text-sm font-medium text-red-900">
          {profileError?.message || "Failed to load settings"}
        </p>
        <Button variant="outline" onClick={() => refetchProfile()}>
          Try again
        </Button>
      </div>
    );
  }

  if (!isOverview && activeDetailMeta) {
    const SummaryIcon = activeSectionSummary?.icon ?? null;

    return (
      <div className="space-y-8">
        <div>
          <Button
            variant="ghost"
            size="sm"
            onClick={goBack}
            className="group inline-flex items-center gap-2 px-0 text-slate-600 hover:text-slate-900"
          >
            <ArrowLeft className="h-4 w-4 transition-transform group-hover:-translate-x-0.5" />
            Back
          </Button>
        </div>

        <div className="space-y-2">
          <h1 className="text-3xl font-semibold text-slate-900">
            {activeDetailMeta.title}
          </h1>
          <p className="text-sm text-slate-600">
            {activeDetailMeta.description}
          </p>
        </div>

        {activeSectionSummary && SummaryIcon ? (
          <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
            <div className="flex items-start gap-4">
              <div className={`flex h-10 w-10 items-center justify-center rounded-full ${TILE_TONES[currentView].tile}`}>
                <SummaryIcon className={`h-5 w-5 ${TILE_TONES[currentView].icon}`} />
              </div>
              <div>
                <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">
                  {activeSectionSummary.title}
                </p>
                <p className="mt-2 text-lg font-semibold text-slate-900">
                  {activeSectionSummary.value}
                </p>
                <p className="mt-1 text-sm text-slate-600">
                  {activeSectionSummary.helper}
                </p>
              </div>
            </div>
          </div>
        ) : null}

        <div className="space-y-6">
          {renderDetailContent()}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      <div className="space-y-2">
        <h1 className="text-3xl font-semibold text-slate-900">Workspace settings</h1>
        <p className="text-sm text-slate-600">
          Open a section below to review the full details without the clutter.
        </p>
      </div>

      <PlaybookSetupCard />

      <div className="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <ul className="divide-y divide-slate-100">
          {overviewSections.map((section) => {
            const SectionIcon = section.icon;
            const isDisabled = Boolean(section.disabled);
            const tone = TILE_TONES[section.key];

            return (
              <li key={section.key}>
                <button
                  type="button"
                  onClick={() => {
                    if (!isDisabled) {
                      pushView(section.key);
                    }
                  }}
                  disabled={isDisabled}
                  className={`flex w-full items-start justify-between gap-6 px-6 py-5 text-left transition ${
                    isDisabled
                      ? "cursor-not-allowed opacity-60"
                      : "hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-slate-900/10"
                  }`}
                >
                  <div className="flex items-start gap-4">
                    <div className={`flex h-10 w-10 items-center justify-center rounded-full ${tone.tile}`}>
                      <SectionIcon className={`h-5 w-5 ${tone.icon}`} />
                    </div>
                    <div className="space-y-1">
                      <p className="text-sm font-semibold text-slate-900">
                        {section.title}
                      </p>
                      <p className="text-sm text-slate-600">
                        {section.description}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    <div className="text-right">
                      <p className="text-base font-semibold text-slate-900">
                        {section.value}
                      </p>
                      <p className="mt-1 text-xs text-slate-500">
                        {section.helper}
                      </p>
                    </div>
                    <ChevronRight className="h-4 w-4 text-slate-400" aria-hidden="true" />
                  </div>
                </button>
              </li>
            );
          })}
        </ul>
      </div>
    </div>
  );
}
