"use client";

import { useMemo, useState, useTransition } from "react";
import { format } from "date-fns";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import {
  AlertTriangle,
  CalendarDays,
  CheckCircle2,
  Clock,
  LifeBuoy,
  Loader2,
  RefreshCcw,
  Sparkles,
} from "lucide-react";
import { toast } from "sonner";

import { PlansModal } from "@/components/billing/plans-modal";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import type { SubscriptionGateState } from "@/lib/polar/current-subscription";
import { getPlanById, getPlanByProductId } from "@/lib/polar/plans";
import { useProductsQuery } from "@/lib/hooks/queries/use-products-query";
import { useAiUsageQuery } from "@/lib/hooks/queries/use-ai-usage-query";
import { cancelSubscription } from "@/lib/actions/billing/cancel-subscription";
import { cancelMPSubscription } from "@/lib/actions/billing/cancel-mp-subscription";
import { isMercadoPagoEnabled } from "@/lib/mercadopago/config";
import { ui, tpl } from "@/lib/copy/ui";

interface SubscriptionTabProps {
  state: SubscriptionGateState | null;
  isLoading: boolean;
  error: string | null;
  onRefresh: () => Promise<void>;
}

export function SubscriptionTab({
  state,
  isLoading,
  error,
  onRefresh,
}: SubscriptionTabProps) {
  const [isPlansOpen, setPlansOpen] = useState(false);
  const [isPlanChangePending, setPlanChangePending] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [actionState, setActionState] = useState<"idle" | "cancelling" | "resuming">(
    "idle"
  );
  const [isCancelDialogOpen, setCancelDialogOpen] = useState(false);
  const [cancelInput, setCancelInput] = useState("");
  const [isPending, startTransition] = useTransition();

  const billingConfigured = state?.reason !== "POLAR_UNCONFIGURED" && state?.reason !== "MP_UNCONFIGURED";
  const isActive = Boolean(state?.isActive);
  const showInactive = !isActive || state?.reason === "NO_ACTIVE_SUBSCRIPTION";
  const canInteract = billingConfigured && !isPlanChangePending && actionState === "idle";
  const mercadopagoEnabled = isMercadoPagoEnabled();

  const { data: products } = useProductsQuery();
  const { data: aiUsage } = useAiUsageQuery(!showInactive);

  const aiCreditsPercent = aiUsage && aiUsage.credits_max > 0
    ? Math.min(100, Math.round((aiUsage.credits_used / aiUsage.credits_max) * 100))
    : 0;

  const plan = useMemo(() => {
    if (!state || !products) return null;
    if (state.planId) {
      const byId = getPlanById(products, state.planId);
      if (byId) return byId;
    }
    return getPlanByProductId(products, state.subscription?.productId ?? null);
  }, [state, products]);

  const usage = state?.usage;
  const includedInvoices = usage?.included ?? plan?.includedInvoices ?? 0;
  const usedInvoices = usage?.used ?? 0;
  const usagePercent =
    includedInvoices > 0
      ? Math.min(100, Math.round((usedInvoices / includedInvoices) * 100))
      : 0;

  const nextBillingDate = state?.subscription?.currentPeriodEnd
    ? new Date(state.subscription.currentPeriodEnd)
    : null;
  const trialEndDate = state?.subscription?.trialEnd
    ? new Date(state.subscription.trialEnd)
    : null;
  const cancelAtPeriodEnd = state?.subscription?.cancelAtPeriodEnd ?? false;

  const statusDisplay = getStatusDisplay(state, cancelAtPeriodEnd);
  const planPrice =
    plan?.price != null
      ? new Intl.NumberFormat(undefined, {
          style: "currency",
          currency: "USD",
        }).format(plan.price)
      : null;

  if (isLoading && !state) {
    return <SubscriptionSkeleton />;
  }

  const contactHref =
    process.env.NEXT_PUBLIC_CONTACT_EMAIL ||
    process.env.NOTIFICATION_EMAIL ||
    process.env.NOTIFICATION_EMAIL ||
    "mailto:info@yourdomain.com";

  if (!billingConfigured) {
    return (
      <Card className="border-amber-200 bg-amber-50/60">
        <CardHeader className="flex items-start gap-3">
          <AlertTriangle className="mt-1 h-5 w-5 text-amber-500" />
          <div className="space-y-1">
            <CardTitle className="text-lg text-amber-900">
              {ui.billing.polarConfigRequired}
            </CardTitle>
            <CardDescription className="text-sm text-amber-800">
              {ui.billing.polarConfigDesc}
            </CardDescription>
          </div>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-3">
          <Button
            variant="outline"
            onClick={() => {
              void onRefresh();
            }}
          >
            <RefreshCcw className="mr-2 h-4 w-4" />
            {ui.billing.checkAgain}
          </Button>
          <Button variant="ghost" asChild className="text-amber-800 hover:text-amber-900">
            <a href={contactHref}>{ui.billing.talkToSupport}</a>
          </Button>
        </CardContent>
      </Card>
    );
  }

  const overlayActive = isPlanChangePending || actionState !== "idle";

  const summarySection = showInactive ? (
    <section className="rounded-3xl border border-dashed border-gray-300 bg-white p-10 text-center shadow-sm">
      <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-gray-100">
        <AlertTriangle className="h-6 w-6 text-gray-500" />
      </div>
      <h3 className="mt-6 text-2xl font-semibold text-gray-900">{ui.billing.noActivePlan}</h3>
      <p className="mt-2 text-sm text-gray-600">
        {ui.billing.noActivePlanBody}
      </p>
      <div className="mt-6 flex flex-col items-center justify-center gap-3 sm:flex-row">
        <Button
          onClick={() => setPlansOpen(true)}
          disabled={!canInteract}
          className="bg-gray-900 text-white hover:bg-gray-800"
        >
          {ui.billing.browsePlans}
        </Button>
        <Button variant="outline" asChild>
          <a href={contactHref} className="text-sm">
            {ui.billing.talkToSales}
          </a>
        </Button>
      </div>
    </section>
  ) : (
    <section className="rounded-3xl border border-gray-200 bg-white p-8 shadow-sm">
      <div className="flex flex-col gap-6 lg:flex-row lg:items-start lg:justify-between">
        <div className="space-y-2">
          <div className="inline-flex items-center gap-2 rounded-full bg-gray-100 px-3 py-1 text-xs font-semibold uppercase tracking-[0.2em] text-gray-600">
            <CalendarDays className="h-3.5 w-3.5" />
            {ui.billing.currentPlanEyebrow}
          </div>
          <h3 className="text-3xl font-semibold text-gray-900">
            {plan?.name ?? ui.billing.customPlan}
          </h3>
          <p className="text-sm text-gray-600">
            {planPrice ? `${planPrice} • ${ui.billing.billedMonthly}` : `${ui.billing.billedVia} ${mercadopagoEnabled ? "MercadoPago" : "Polar"}`}
          </p>
        </div>
        <Badge className={statusDisplay.className}>{statusDisplay.label}</Badge>
      </div>

      <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {nextBillingDate && (
          <SummaryMetric label={ui.billing.renewsOn} value={format(nextBillingDate, "MMM d, yyyy")} />
        )}
        {trialEndDate && (
          <SummaryMetric label={ui.billing.trialEnds} value={format(trialEndDate, "MMM d, yyyy")} />
        )}
        {plan?.includedInvoices != null && (
          <SummaryMetric
            label={ui.billing.invoicesPerMonth}
            value={plan.includedInvoices.toLocaleString()}
          />
        )}
        {plan?.includedSeats != null && (
          <SummaryMetric
            label={ui.billing.seatsIncluded}
            value={plan.includedSeats.toLocaleString()}
          />
        )}
      </div>

      <div className="mt-6 space-y-3">
        {cancelAtPeriodEnd && nextBillingDate ? (
          <Alert className="border border-amber-200 bg-amber-50">
            <AlertTitle className="text-sm font-semibold text-amber-900">
              {ui.billing.scheduledToEnd}
            </AlertTitle>
            <AlertDescription className="text-sm text-amber-800">
              {tpl(ui.billing.scheduledToEndBody, {
                date: format(nextBillingDate, "MMM d, yyyy"),
              })}
            </AlertDescription>
          </Alert>
        ) : (
          <p className="text-sm text-gray-600">
            {ui.billing.switchPlansHint}
          </p>
        )}

        {plan?.benefits?.length ? (
          <div className="rounded-2xl border border-dashed border-gray-200 bg-gray-50 p-5">
            <p className="text-xs font-semibold uppercase tracking-[0.2em] text-gray-500">
              {ui.billing.planBenefits}
            </p>
            <ul className="mt-3 space-y-2">
              {plan.benefits.map((benefit) => (
                <li key={benefit} className="flex items-start gap-2 text-sm text-gray-600">
                  <CheckCircle2 className="mt-0.5 h-4 w-4 text-gray-500" />
                  <span>{benefit}</span>
                </li>
              ))}
            </ul>
          </div>
        ) : null}

        <div className="flex flex-wrap items-center gap-3">
          {cancelAtPeriodEnd ? (
            <Button
              variant="outline"
              onClick={() => handleUpdateCancellation(false)}
              disabled={actionState !== "idle"}
            >
              {ui.billing.resumeSubscription}
              </Button>
          ) : (
            <Button
              variant="outline"
              className="border-red-200 text-red-600 hover:bg-red-50"
              onClick={() => setCancelDialogOpen(true)}
              disabled={actionState !== "idle"}
            >
              {ui.billing.scheduleCancellation}
            </Button>
          )}
          <Button
            variant="outline"
            onClick={() => {
              setPlanChangePending(true);
              void onRefresh().finally(() => setPlanChangePending(false));
            }}
            disabled={!canInteract}
          >
            <RefreshCcw className="mr-2 h-4 w-4" />
            {ui.billing.refreshStatus}
          </Button>
          <Button variant="ghost" asChild>
            <a href={contactHref} className="text-sm text-gray-600 hover:text-gray-900">
              <LifeBuoy className="mr-2 h-4 w-4" />
              {ui.common.contactSupport}
            </a>
          </Button>
        </div>
      </div>
    </section>
  );

  const usageSection =
    !showInactive && includedInvoices > 0 ? (
      <section className="rounded-3xl border border-gray-200 bg-white p-8 shadow-sm">
        <div className="flex items-start justify-between gap-4">
          <div className="space-y-1">
            <h4 className="text-lg font-semibold text-gray-900">{ui.billing.invoiceUsage}</h4>
            <p className="text-sm text-gray-600">
              {nextBillingDate
                ? tpl(ui.billing.usageResetsOn, { date: format(nextBillingDate, "MMM d, yyyy") })
                : ui.billing.trackUsage}
            </p>
          </div>
          <Badge className="bg-gray-100 text-gray-700">
            {usagePercent}% of {includedInvoices.toLocaleString()}
          </Badge>
        </div>
        <div className="mt-6 space-y-3">
          <Progress value={usagePercent} className="h-2" />
          <div className="flex flex-wrap items-center gap-4 text-sm text-gray-600">
            <span>
              <span className="font-semibold text-gray-900">
                {usedInvoices.toLocaleString()}
              </span>{" "}
              {ui.billing.invoicesProcessed}
            </span>
            <span className="flex items-center gap-2 text-xs text-gray-500">
              <Clock className="h-4 w-4" />
              {remainingInvoicesText(includedInvoices, usedInvoices)}
            </span>
          </div>
        </div>
      </section>
    ) : null;

  const aiUsageSection =
    !showInactive && aiUsage && aiUsage.credits_max > 0 ? (
      <section className="rounded-3xl border border-gray-200 bg-white p-8 shadow-sm">
        <div className="flex items-start justify-between gap-4">
          <div className="space-y-1">
            <h4 className="text-lg font-semibold text-gray-900">{ui.billing.aiCredits}</h4>
            <p className="text-sm text-gray-600">
              {tpl(ui.billing.aiCreditsBody, {
                resets: aiUsage.period_end
                  ? tpl(ui.billing.aiResetsOn, {
                      date: format(new Date(aiUsage.period_end), "MMM d, yyyy"),
                    })
                  : "",
              })}
            </p>
          </div>
          <Badge className="bg-gray-100 text-gray-700">
            {aiUsage.credits_used.toLocaleString()} / {aiUsage.credits_max.toLocaleString()}
          </Badge>
        </div>
        <div className="mt-6 space-y-3">
          <Progress value={aiCreditsPercent} className="h-2" />
          <div className="flex flex-wrap items-center gap-4 text-sm text-gray-600">
            <span>
              <span className="font-semibold text-gray-900">
                {aiUsage.credits_remaining.toLocaleString()}
              </span>{" "}
              {ui.billing.creditsRemaining}
            </span>
            <span className="flex items-center gap-2 text-xs text-gray-500">
              <Sparkles className="h-4 w-4" />
              {aiUsage.tokens_input.toLocaleString()} / {aiUsage.tokens_output.toLocaleString()}{" "}
              {ui.billing.inOutTokens}
            </span>
          </div>
        </div>
      </section>
    ) : null;

  return (
    <>
      <div className="relative">
        {overlayActive && (
          <div className="absolute inset-0 z-20 rounded-3xl bg-white/70 backdrop-blur-sm">
            <div className="flex h-full flex-col items-center justify-center gap-3 text-sm font-medium text-gray-700">
              <Loader2 className="h-6 w-6 animate-spin" />
              {ui.billing.processingRequest}
            </div>
          </div>
        )}

        <div className="space-y-8">
          {error ? (
            <Alert variant="destructive">
              <AlertTitle>{ui.billing.unableLoadSubscription}</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : null}

          {actionError ? (
            <Alert variant="destructive">
              <AlertTitle>{ui.billing.actionFailed}</AlertTitle>
              <AlertDescription>{actionError}</AlertDescription>
            </Alert>
          ) : null}

          {summarySection}
          {usageSection}
          {aiUsageSection}
        </div>
      </div>

      <PlansModal
        open={isPlansOpen}
        onOpenChange={(open) => {
          if (!open) {
            setPlanChangePending(false);
          }
          setPlansOpen(open);
        }}
        subscriptionState={state}
        onPlanChangePending={(pending) => setPlanChangePending(pending)}
        mercadopagoEnabled={mercadopagoEnabled}
      />

      <Dialog
        open={isCancelDialogOpen}
        onOpenChange={(open) => {
          if (actionState === "cancelling") return;
          if (!open) {
            setCancelInput("");
          }
          setCancelDialogOpen(open);
        }}
      >
        <DialogContent className="max-w-lg">
          <DialogHeader className="text-left">
            <DialogTitle className="text-xl font-semibold text-gray-900">
              {ui.billing.confirmCancellation}
            </DialogTitle>
            <DialogDescription className="text-sm text-gray-600">
              {tpl(ui.billing.cancelDialogBody, {
                date: nextBillingDate
                  ? format(nextBillingDate, "MMM d, yyyy")
                  : ui.billing.endOfTerm,
              })}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <Input
              value={cancelInput}
              onChange={(event) => setCancelInput(event.target.value)}
              placeholder={ui.billing.typeCancelPlaceholder}
              className="uppercase tracking-[0.3em]"
              autoFocus
            />
            <Alert className="border border-amber-200 bg-amber-50">
              <AlertTitle className="text-sm font-semibold text-amber-900">
                {ui.billing.headsUp}
              </AlertTitle>
              <AlertDescription className="text-sm text-amber-800">
                {ui.billing.headsUpBody}
              </AlertDescription>
            </Alert>
          </div>
          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-between sm:gap-3">
            <Button
              variant="outline"
              onClick={() => {
                setCancelInput("");
                setCancelDialogOpen(false);
              }}
              disabled={actionState === "cancelling"}
            >
              {ui.billing.keepSubscription}
            </Button>
            <Button
              variant="destructive"
              onClick={() => handleUpdateCancellation(true)}
              disabled={cancelInput.trim().toUpperCase() !== "CANCEL" || actionState === "cancelling"}
            >
              {actionState === "cancelling" ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : null}
              {ui.billing.confirmCancellation}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );

  async function handleUpdateCancellation(cancel: boolean) {
    if (!state?.subscription || actionState !== "idle") {
      return;
    }

    setActionError(null);
    setActionState(cancel ? "cancelling" : "resuming");

    const subscriptionId = state.subscription.id;

    startTransition(async () => {
      try {
        const result =
          cancel && mercadopagoEnabled
            ? await cancelMPSubscription({ subscriptionId })
            : await cancelSubscription({ cancelAtPeriodEnd: cancel });

        if (!result.success) {
          throw new Error(result.error);
        }

        if (cancel) {
          toast.success("Cancellation scheduled successfully.");
          setCancelInput("");
          setCancelDialogOpen(false);
        } else {
          toast.success("Subscription resumed — you will remain active.");
        }

        await onRefresh();
      } catch (updateError) {
        console.error("[Settings] Failed to update subscription cancellation", updateError);
        const message =
          updateError instanceof Error
            ? updateError.message
            : "We could not update your subscription. Please try again.";
        setActionError(message);
        toast.error(message);
      } finally {
        setActionState("idle");
      }
    });
  }
}

function SummaryMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-gray-100 bg-gray-50 px-4 py-3 text-left">
      <p className="text-xs font-semibold uppercase tracking-[0.2em] text-gray-500">
        {label}
      </p>
      <p className="mt-1 text-sm font-semibold text-gray-900">{value}</p>
    </div>
  );
}

function remainingInvoicesText(limit: number, used: number) {
  const remaining = Math.max(limit - used, 0);
  return `${remaining.toLocaleString()} ${ui.billing.invoicesRemainingSuffix}`;
}

function SubscriptionSkeleton() {
  return (
    <Card className="border-gray-200">
      <CardHeader>
        <CardTitle className="text-xl">{ui.billing.subscriptionOverview}</CardTitle>
        <CardDescription className="text-sm text-gray-600">
          {ui.billing.loadingPlanDetails}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <Skeleton className="h-12 w-44 rounded-2xl" />
        <Skeleton className="h-32 w-full rounded-3xl" />
        <Skeleton className="h-20 w-full rounded-3xl" />
      </CardContent>
    </Card>
  );
}

function getStatusDisplay(
  state: SubscriptionGateState | null | undefined,
  cancelAtPeriodEnd: boolean
) {
  const status = state?.status ?? null;
  if (!state?.isActive) {
    return {
      label: status ? titleCase(status) : ui.billing.statusInactive,
      className: "bg-gray-200 text-gray-700",
    };
  }

  if (cancelAtPeriodEnd) {
    return {
      label: ui.billing.statusCancelsSoon,
      className: "bg-amber-100 text-amber-700",
    };
  }

  switch (status) {
    case "trialing":
      return {
        label: ui.billing.statusTrialing,
        className: "bg-blue-100 text-blue-700",
      };
    case "past_due":
      return {
        label: ui.billing.statusPastDue,
        className: "bg-amber-100 text-amber-700",
      };
    case "grace":
      return {
        label: ui.billing.statusGrace,
        className: "bg-amber-100 text-amber-700",
      };
    case "active":
    default:
      return {
        label: status ? titleCase(status) : ui.billing.statusActive,
        className: "bg-emerald-100 text-emerald-700",
      };
  }
}

function titleCase(value: string) {
  return value
    .split("_")
    .map((chunk) => chunk.charAt(0).toUpperCase() + chunk.slice(1))
    .join(" ");
}
