"use client";

import { useEffect, useMemo, useState, useTransition } from "react";
import { format } from "date-fns";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import {
  AlertTriangle,
  CalendarDays,
  CheckCircle2,
  CircleOff,
  Clock,
  LifeBuoy,
  Loader2,
  RefreshCcw,
  Sparkles,
} from "lucide-react";
import { toast } from "sonner";

import { PlansModal } from "@/components/billing/plans-modal";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
import { StatusChip } from "@/components/ui/status-chip";
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

  const searchParams = useSearchParams();
  const pathname = usePathname();
  const router = useRouter();

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

  // Amber threshold (design language): usage >= 80% of the plan limit.
  const USAGE_THRESHOLD = 80;
  const usageAtLimit = usagePercent >= USAGE_THRESHOLD;
  const aiUsageAtLimit = aiCreditsPercent >= USAGE_THRESHOLD;

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

  // Drop checkout-outcome params once the user acknowledges the banner so it
  // does not reappear on refresh. Preserves any other params (e.g. view).
  const clearPaymentParams = () => {
    const params = new URLSearchParams(searchParams.toString());
    params.delete("payment_verified");
    params.delete("payment_error");
    const query = params.toString();
    router.replace(query ? `${pathname}?${query}` : pathname);
  };

  // Acknowledge checkout-outcome params after the banner renders (not only on
  // dismiss): strip them from the URL via history.replaceState so refresh and
  // back-navigation do not re-show a stale banner. Also refetch the
  // subscription state once so webhook/verify-driven changes appear without a
  // manual "Actualizar estado" click.
  useEffect(() => {
    const hadOutcome =
      searchParams.get("payment_verified") === "true" ||
      searchParams.get("payment_error") === "true";

    if (hadOutcome) {
      const params = new URLSearchParams(window.location.search);
      params.delete("payment_verified");
      params.delete("payment_error");
      const query = params.toString();
      window.history.replaceState(
        null,
        "",
        query ? `${window.location.pathname}?${query}` : window.location.pathname
      );
      void onRefresh();
    }
    // Run once on mount: the checkout callback is a fresh page navigation.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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
              {state?.reason === "MP_UNCONFIGURED"
                ? ui.billing.mpConfigRequired
                : ui.billing.polarConfigRequired}
            </CardTitle>
            <CardDescription className="text-sm text-amber-800">
              {state?.reason === "MP_UNCONFIGURED"
                ? ui.billing.mpConfigDesc
                : ui.billing.polarConfigDesc}
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
    <section className="rounded-3xl border border-dashed border-slate-300 bg-white p-10 text-center shadow-sm">
      <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-slate-100">
        <AlertTriangle className="h-6 w-6 text-slate-500" />
      </div>
      <h3 className="mt-6 text-2xl font-semibold text-slate-900">{ui.billing.noActivePlan}</h3>
      <p className="mt-2 text-sm text-slate-600">
        {ui.billing.noActivePlanBody}
      </p>
      <div className="mt-6 flex flex-col items-center justify-center gap-3 sm:flex-row">
        <Button
          onClick={() => setPlansOpen(true)}
          disabled={!canInteract}
          className="bg-emerald-500 text-white hover:bg-emerald-600"
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
    <section className="rounded-3xl border border-slate-200 bg-white p-8 shadow-sm">
      <div className="flex flex-col gap-6 lg:flex-row lg:items-start lg:justify-between">
        <div className="space-y-2">
          <div className="inline-flex items-center gap-2 rounded-full bg-slate-100 px-3 py-1 text-xs font-semibold uppercase tracking-[0.2em] text-slate-600">
            <CalendarDays className="h-3.5 w-3.5" />
            {ui.billing.currentPlanEyebrow}
          </div>
          <h3 className="text-3xl font-semibold text-slate-900">
            {plan?.name ?? ui.billing.customPlan}
          </h3>
          <p className="text-sm text-slate-600">
            {planPrice ? `${planPrice} • ${ui.billing.billedMonthly}` : `${ui.billing.billedVia} ${mercadopagoEnabled ? "MercadoPago" : "Polar"}`}
          </p>
        </div>
        <StatusChip tone={statusDisplay.tone} icon={statusDisplay.icon}>
          {statusDisplay.label}
        </StatusChip>
      </div>

      {/* Grace-period messaging: features stay readable, writes are blocked.
          past_due with IsGracePeriod = the subscription hasn't been cancelled yet. */}
      {isActive && state?.status === "past_due" && (
        <Alert variant="default" className="mt-4 border-amber-200 bg-amber-50">
          <AlertTriangle className="h-4 w-4 text-amber-600" />
          <AlertTitle className="text-amber-800">{ui.billing.gracePeriodTitle}</AlertTitle>
          <AlertDescription className="text-amber-700">
            {ui.billing.gracePeriodBody}
          </AlertDescription>
        </Alert>
      )}

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
              {tpl(
                mercadopagoEnabled
                  ? ui.billing.scheduledToEndBodyMp
                  : ui.billing.scheduledToEndBody,
                { date: format(nextBillingDate, "MMM d, yyyy") }
              )}
            </AlertDescription>
          </Alert>
        ) : (
          <p className="text-sm text-slate-600">
            {ui.billing.switchPlansHint}
          </p>
        )}

        {plan?.benefits?.length ? (
          <div className="rounded-2xl border border-dashed border-slate-200 bg-slate-50 p-5">
            <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">
              {ui.billing.planBenefits}
            </p>
            <ul className="mt-3 space-y-2">
              {plan.benefits.map((benefit) => (
                <li key={benefit} className="flex items-start gap-2 text-sm text-slate-600">
                  <CheckCircle2 className="mt-0.5 h-4 w-4 text-slate-500" />
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
            <a href={contactHref} className="text-sm text-slate-600 hover:text-slate-900">
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
      <section className="rounded-3xl border border-slate-200 bg-white p-8 shadow-sm">
        <div className="flex items-start justify-between gap-4">
          <div className="space-y-1">
            <h4 className="text-lg font-semibold text-slate-900">{ui.billing.invoiceUsage}</h4>
            <p className="text-sm text-slate-600">
              {nextBillingDate
                ? tpl(ui.billing.usageResetsOn, { date: format(nextBillingDate, "MMM d, yyyy") })
                : ui.billing.trackUsage}
            </p>
          </div>
          <StatusChip tone={usageAtLimit ? "amber" : "gray"} icon={usageAtLimit ? AlertTriangle : undefined}>
            {usageAtLimit
              ? tpl(ui.settings.usageAtLimit, { percent: String(usagePercent) })
              : tpl(ui.settings.usagePercent, { percent: String(usagePercent) })}
          </StatusChip>
        </div>
        <div className="mt-6 space-y-3">
          <Progress
            value={usagePercent}
            className={usageAtLimit ? "h-2 [&>div]:bg-amber-500" : "h-2"}
          />
          <div className="flex flex-wrap items-center gap-4 text-sm text-slate-600">
            <span>
              <span className="font-semibold text-slate-900">
                {usedInvoices.toLocaleString()}
              </span>{" "}
              {ui.billing.invoicesProcessed}
            </span>
            <span className="flex items-center gap-2 text-xs text-slate-500">
              <Clock className="h-4 w-4" />
              {remainingInvoicesText(includedInvoices, usedInvoices)}
            </span>
          </div>
          {usageAtLimit ? (
            <p className="text-xs font-medium text-amber-700">{ui.settings.usageNearLimitBody}</p>
          ) : null}
        </div>
      </section>
    ) : null;

  const aiUsageSection =
    !showInactive && aiUsage && aiUsage.credits_max > 0 ? (
      <section className="rounded-3xl border border-slate-200 bg-white p-8 shadow-sm">
        <div className="flex items-start justify-between gap-4">
          <div className="space-y-1">
            <h4 className="text-lg font-semibold text-slate-900">{ui.billing.aiCredits}</h4>
            <p className="text-sm text-slate-600">
              {tpl(ui.billing.aiCreditsBody, {
                resets: aiUsage.period_end
                  ? tpl(ui.billing.aiResetsOn, {
                      date: format(new Date(aiUsage.period_end), "MMM d, yyyy"),
                    })
                  : "",
              })}
            </p>
          </div>
          <StatusChip tone={aiUsageAtLimit ? "amber" : "gray"} icon={aiUsageAtLimit ? AlertTriangle : undefined}>
            {aiUsageAtLimit
              ? tpl(ui.settings.usageAtLimit, { percent: String(aiCreditsPercent) })
              : `${aiUsage.credits_used.toLocaleString()} / ${aiUsage.credits_max.toLocaleString()}`}
          </StatusChip>
        </div>
        <div className="mt-6 space-y-3">
          <Progress
            value={aiCreditsPercent}
            className={aiUsageAtLimit ? "h-2 [&>div]:bg-amber-500" : "h-2"}
          />
          <div className="flex flex-wrap items-center gap-4 text-sm text-slate-600">
            <span>
              <span className="font-semibold text-slate-900">
                {aiUsage.credits_remaining.toLocaleString()}
              </span>{" "}
              {ui.billing.creditsRemaining}
            </span>
            <span className="flex items-center gap-2 text-xs text-slate-500">
              <Sparkles className="h-4 w-4" />
              {aiUsage.tokens_input.toLocaleString()} / {aiUsage.tokens_output.toLocaleString()}{" "}
              {ui.billing.inOutTokens}
            </span>
          </div>
          {aiUsageAtLimit ? (
            <p className="text-xs font-medium text-amber-700">{ui.settings.usageNearLimitBody}</p>
          ) : null}
        </div>
      </section>
    ) : null;

  return (
    <>
      <div className="relative">
        {overlayActive && (
          <div className="absolute inset-0 z-20 rounded-3xl bg-white/70 backdrop-blur-sm">
            <div className="flex h-full flex-col items-center justify-center gap-3 text-sm font-medium text-slate-700">
              <Loader2 className="h-6 w-6 animate-spin" />
              {ui.billing.processingRequest}
            </div>
          </div>
        )}

        <div className="space-y-8">
          {searchParams.get("payment_verified") === "true" ? (
            <Alert className="border border-emerald-200 bg-emerald-50">
              <CheckCircle2 className="h-4 w-4 text-primary" />
              <AlertTitle className="flex items-center gap-2">
                {ui.billing.paymentVerifiedTitle}
              </AlertTitle>
              <AlertDescription className="text-emerald-700">
                {ui.billing.paymentVerifiedBody}
              </AlertDescription>
              <div className="mt-3 flex justify-end">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={clearPaymentParams}
                  className="border-emerald-200 bg-white text-emerald-700 hover:bg-emerald-50 hover:text-emerald-800"
                >
                  {ui.billing.understood}
                </Button>
              </div>
            </Alert>
          ) : null}

          {searchParams.get("payment_error") === "true" ? (
            <Alert variant="destructive">
              <AlertTriangle className="h-4 w-4" />
              <AlertTitle>{ui.billing.paymentErrorTitle}</AlertTitle>
              <AlertDescription>{ui.billing.paymentErrorBody}</AlertDescription>
              <div className="mt-3 flex flex-wrap gap-2">
                <Button size="sm" variant="destructive" onClick={() => setPlansOpen(true)}>
                  <RefreshCcw className="mr-2 h-4 w-4" />
                  {ui.billing.retryCheckout}
                </Button>
                <Button size="sm" variant="outline" onClick={clearPaymentParams}>
                  {ui.billing.understood}
                </Button>
              </div>
            </Alert>
          ) : null}

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
            <DialogTitle className="text-xl font-semibold text-slate-900">
              {ui.billing.confirmCancellation}
            </DialogTitle>
            <DialogDescription className="text-sm text-slate-600">
              {mercadopagoEnabled
                ? ui.billing.cancelDialogBodyMp
                : tpl(ui.billing.cancelDialogBody, {
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
                {mercadopagoEnabled ? ui.billing.headsUpBodyMp : ui.billing.headsUpBody}
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
        // Provider-accurate branching: under MercadoPago both cancel and
        // resume route through the MP action (Polar's cancelSubscription
        // errors with "No active subscription to update." for MP orgs); Polar
        // keeps its own cancel/resume path.
        const result = mercadopagoEnabled
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
    <div className="rounded-2xl border border-slate-100 bg-slate-50 px-4 py-3 text-left">
      <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">
        {label}
      </p>
      <p className="mt-1 text-sm font-semibold text-slate-900">{value}</p>
    </div>
  );
}

function remainingInvoicesText(limit: number, used: number) {
  const remaining = Math.max(limit - used, 0);
  return `${remaining.toLocaleString()} ${ui.billing.invoicesRemainingSuffix}`;
}

function SubscriptionSkeleton() {
  return (
    <Card className="border-slate-200">
      <CardHeader>
        <CardTitle className="text-xl">{ui.billing.subscriptionOverview}</CardTitle>
        <CardDescription className="text-sm text-slate-600">
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
      tone: "gray" as const,
      icon: CircleOff,
    };
  }

  if (cancelAtPeriodEnd) {
    return {
      label: ui.billing.statusCancelsSoon,
      tone: "amber" as const,
      icon: AlertTriangle,
    };
  }

  switch (status) {
    case "trialing":
      return {
        label: ui.billing.statusTrialing,
        tone: "blue" as const,
        icon: Sparkles,
      };
    case "past_due":
      return {
        label: ui.billing.statusPastDue,
        tone: "amber" as const,
        icon: AlertTriangle,
      };
    case "grace":
      return {
        label: ui.billing.statusGrace,
        tone: "amber" as const,
        icon: Clock,
      };
    case "active":
    default:
      return {
        label: status ? titleCase(status) : ui.billing.statusActive,
        tone: "emerald" as const,
        icon: CheckCircle2,
      };
  }
}

function titleCase(value: string) {
  return value
    .split("_")
    .map((chunk) => chunk.charAt(0).toUpperCase() + chunk.slice(1))
    .join(" ");
}
