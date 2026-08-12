"use client";

import { useMemo } from "react";
import Link from "next/link";
import { Check, MessageCircle, Sparkles, CreditCard, Inbox, FileText } from "lucide-react";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Button } from "@/components/ui/button";
import { useUIStore } from "@/stores/ui-store";
import { ui, tpl } from "@/lib/copy/ui";
import { useWhatsAppConfigQuery } from "@/lib/hooks/queries/use-whatsapp-config-query";
import { useSubscriptionQuery } from "@/lib/hooks/queries/use-subscription-query";
import { useConversationsQuery } from "@/lib/hooks/queries/use-conversations-query";
import {
  isAssistantIntroDismissed,
  isInboxVisited,
  isKnowledgeVisited,
  loadBusinessContext,
} from "@/lib/onboarding/storage";

type StepKey = "connectWhatsApp" | "choosePlan" | "meetAssistant" | "openInbox" | "addDocument";

interface ChecklistStep {
  key: StepKey;
  title: string;
  description: string;
  done: boolean;
  icon: typeof Check;
  href?: string;
  action?: () => void;
}

export function FirstRunChecklist() {
  const setPlansModalOpen = useUIStore((state) => state.setPlansModalOpen);

  const { data: whatsappConfig } = useWhatsAppConfigQuery();
  const { data: subscriptionState } = useSubscriptionQuery();
  const { data: conversations } = useConversationsQuery();

  const steps = useMemo<ChecklistStep[]>(() => {
    const whatsappConnected = whatsappConfig?.isActive === true;
    const planActive = subscriptionState?.isActive === true;
    const assistantIntroduced = isAssistantIntroDismissed();
    const inboxExplored = isInboxVisited() || (conversations?.length ?? 0) > 0;

    const base: ChecklistStep[] = [];

    // When subscription is inactive, surface the plan choice before the
    // paywalled WhatsApp step so new orgs are not stranded on a raw 402.
    if (!planActive) {
      base.push({
        key: "choosePlan",
        title: ui.onboarding.stepChoosePlan,
        description: ui.onboarding.stepChoosePlanDesc,
        done: false,
        icon: CreditCard,
        action: () => setPlansModalOpen(true),
      });
    }

    base.push({
      key: "connectWhatsApp",
      title: ui.onboarding.stepConnectWhatsApp,
      description: ui.onboarding.stepConnectWhatsAppDesc,
      done: whatsappConnected,
      icon: MessageCircle,
      href: "/dashboard/settings?view=whatsapp",
    });

    if (planActive) {
      base.push({
        key: "choosePlan",
        title: ui.onboarding.stepChoosePlan,
        description: ui.onboarding.stepChoosePlanDesc,
        done: true,
        icon: CreditCard,
        action: () => setPlansModalOpen(true),
      });
    }

    base.push(
      {
        key: "meetAssistant",
        title: ui.onboarding.stepMeetAssistant,
        description: ui.onboarding.stepMeetAssistantDesc,
        done: assistantIntroduced,
        icon: Sparkles,
        href: "/dashboard/settings?view=ai",
      },
      {
        key: "openInbox",
        title: ui.onboarding.stepOpenInbox,
        description: ui.onboarding.stepOpenInboxDesc,
        done: inboxExplored,
        icon: Inbox,
        href: "/dashboard/inbox",
      },
      {
        key: "addDocument",
        title: ui.onboarding.stepAddDocument,
        description: ui.onboarding.stepAddDocumentDesc,
        done: isKnowledgeVisited(),
        icon: FileText,
        href: "/dashboard/knowledge",
      },
    );

    // Client-side business context shapes priority: when the user said they do
    // not have WhatsApp yet, surface the inbox as the first actionable step.
    const context = loadBusinessContext();
    if (context && context.whatsappReadiness !== "already") {
      const [inboxStep] = base.splice(2, 1);
      if (inboxStep) {
        base.unshift(inboxStep);
      }
    }

    return base;
  }, [whatsappConfig, subscriptionState, conversations, setPlansModalOpen]);

  const total = steps.length;
  const completed = steps.filter((step) => step.done).length;

  if (completed >= total) {
    return null;
  }

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-lg text-foreground">{ui.onboarding.checklistTitle}</CardTitle>
        <CardDescription>{ui.onboarding.checklistSubtitle}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        <Progress value={(completed / total) * 100} aria-label={tpl(ui.onboarding.checklistProgress, { done: completed, total })} />
        <p className="text-xs text-muted-foreground">
          {tpl(ui.onboarding.checklistProgress, { done: completed, total })}
        </p>
        <ul className="space-y-1">
          {steps.map((step) => {
            const Icon = step.icon;
            const inner = (
              <div className="flex items-center gap-3 rounded-lg border border-border bg-background px-4 py-3 transition-colors hover:bg-accent">
                <span
                  className={`flex h-7 w-7 flex-none items-center justify-center rounded-full ${
                    step.done
                      ? "bg-primary/10 text-primary"
                      : "bg-muted text-muted-foreground"
                  }`}
                >
                  {step.done ? <Check className="h-4 w-4" aria-hidden /> : <Icon className="h-4 w-4" aria-hidden />}
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block text-sm font-medium text-foreground">{step.title}</span>
                  <span className="block truncate text-xs text-muted-foreground">{step.description}</span>
                </span>
                {step.done ? (
                  <span className="flex-none text-xs font-medium text-primary">{ui.onboarding.statusDone}</span>
                ) : (
                  <span className="flex-none text-xs text-muted-foreground">{ui.onboarding.statusTodo}</span>
                )}
              </div>
            );

            if (step.done) {
              return <li key={step.key}>{inner}</li>;
            }

            if (step.href) {
              return (
                <li key={step.key}>
                  <Link href={step.href} className="block">
                    {inner}
                  </Link>
                </li>
              );
            }

            return (
              <li key={step.key}>
                <Button
                  type="button"
                  variant="ghost"
                  className="h-auto w-full p-0"
                  onClick={step.action}
                >
                  {inner}
                </Button>
              </li>
            );
          })}
        </ul>
      </CardContent>
    </Card>
  );
}
