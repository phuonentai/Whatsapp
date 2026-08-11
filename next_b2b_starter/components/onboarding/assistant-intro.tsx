"use client";

import { useState } from "react";
import Link from "next/link";
import { Bot, X, BookOpen, ArrowRight } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { ui } from "@/lib/copy/ui";
import {
  dismissAssistantIntro,
  isAssistantIntroDismissed,
} from "@/lib/onboarding/storage";

export function AssistantIntro() {
  const [dismissed, setDismissed] = useState(() => isAssistantIntroDismissed());

  if (dismissed) {
    return null;
  }

  const handleDismiss = () => {
    dismissAssistantIntro();
    setDismissed(true);
  };

  return (
    <Card className="border-primary/20 bg-gradient-to-br from-primary/5 to-background">
      <CardHeader className="flex-row items-start justify-between space-y-0 pb-2">
        <div className="flex items-start gap-3">
          <span className="flex h-10 w-10 flex-none items-center justify-center rounded-full bg-primary/10">
            <Bot className="h-5 w-5 text-primary" aria-hidden />
          </span>
          <div className="space-y-1">
            <CardTitle className="text-lg text-foreground">
              {ui.onboarding.assistantIntroTitle}
            </CardTitle>
            <p className="text-sm text-muted-foreground">
              {ui.onboarding.assistantIntroBody}
            </p>
          </div>
        </div>
        <Button
          variant="ghost"
          size="sm"
          className="flex-none px-2"
          onClick={handleDismiss}
          aria-label={ui.onboarding.assistantIntroDismiss}
        >
          <X className="h-4 w-4" aria-hidden />
        </Button>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-start gap-3 rounded-lg border border-border bg-background/60 p-3">
          <BookOpen className="mt-0.5 h-4 w-4 flex-none text-muted-foreground" aria-hidden />
          <div>
            <p className="text-sm font-medium text-foreground">
              {ui.onboarding.assistantIntroKnowledgeTitle}
            </p>
            <p className="text-sm text-muted-foreground">
              {ui.onboarding.assistantIntroKnowledgeBody}
            </p>
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          <Link href="/dashboard/settings?view=ai">
            <Button size="sm">
              {ui.onboarding.assistantIntroCta} <ArrowRight className="ml-2 h-4 w-4" />
            </Button>
          </Link>
          <Button size="sm" variant="outline" onClick={handleDismiss}>
            {ui.onboarding.assistantIntroDismiss}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
