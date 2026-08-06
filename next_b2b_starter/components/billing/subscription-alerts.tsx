"use client";

import Link from "next/link";
import {
  AlertTriangle,
  ShieldAlert,
  Info,
} from "lucide-react";

import { cn } from "@/lib/utils";

export type SubscriptionAlertVariant = "info" | "warning" | "critical";

export interface SubscriptionAlertAction {
  label: string;
  href?: string;
  onClick?: () => void;
  priority?: "primary" | "secondary";
}

export interface SubscriptionAlertDescriptor {
  id: string;
  variant: SubscriptionAlertVariant;
  title: string;
  description: string;
  action?: SubscriptionAlertAction;
  actions?: SubscriptionAlertAction[];
}

interface SubscriptionAlertsProps {
  alerts: SubscriptionAlertDescriptor[];
  className?: string;
}

const variantStyles: Record<
  SubscriptionAlertVariant,
  { container: string; icon: typeof Info }
> = {
  info: {
    container:
      "border border-blue-200 bg-blue-50/70 text-blue-900",
    icon: Info,
  },
  warning: {
    container:
      "border border-amber-200 bg-amber-50/70 text-amber-900",
    icon: AlertTriangle,
  },
  critical: {
    container:
      "border border-red-200 bg-red-50/70 text-red-900",
    icon: ShieldAlert,
  },
};

export function SubscriptionAlerts({ alerts, className }: SubscriptionAlertsProps) {
  if (!alerts.length) {
    return null;
  }

  return (
    <div className={cn("space-y-3", className)}>
      {alerts.map((alert) => {
        const { container, icon: Icon } = variantStyles[alert.variant];
        const actions =
          alert.actions ??
          (alert.action ? [alert.action] : []);

        return (
          <div
            key={alert.id}
            className={cn(
              "flex gap-3 rounded-2xl px-4 py-3 shadow-sm transition",
              container
            )}
          >
            <Icon className="mt-1 h-5 w-5 flex-none" />
            <div className="flex-1 space-y-2">
              <p className="text-sm font-semibold leading-tight">
                {alert.title}
              </p>
              <p className="text-sm leading-relaxed text-gray-700">
                {alert.description}
              </p>
              {actions.length ? (
                <div className="flex flex-wrap gap-2">
                  {actions.map((action) => {
                    const baseClasses = cn(
                      "inline-flex items-center rounded-md px-3 py-2 text-sm font-semibold transition focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2",
                      action.priority === "primary"
                        ? "bg-gray-900 text-white hover:bg-gray-800 focus-visible:outline-gray-900"
                        : "border border-gray-300 bg-white text-gray-900 hover:bg-gray-100 focus-visible:outline-gray-300"
                    );

                    if (action.onClick) {
                      return (
                        <button
                          key={`${alert.id}-${action.label}`}
                          type="button"
                          className={baseClasses}
                          onClick={action.onClick}
                        >
                          {action.label}
                        </button>
                      );
                    }

                    if (action.href) {
                      return (
                        <Link
                          key={`${alert.id}-${action.label}`}
                          href={action.href}
                          className={baseClasses}
                        >
                          {action.label}
                        </Link>
                      );
                    }

                    return null;
                  })}
                </div>
              ) : null}
            </div>
          </div>
        );
      })}
    </div>
  );
}
