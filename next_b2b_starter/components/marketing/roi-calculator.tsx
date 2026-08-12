"use client";

import { useState } from "react";
import Link from "next/link";
import { Clock, TrendingUp } from "lucide-react";
import { Button } from "@/components/ui/button";
import { copy } from "@/lib/copy/ui";

// 0,4 h de digitación por conversación-vendedor × 4,3 semanas/mes × ~$10.000 COP/hora ≈ 1,7 × $10.000 COP.
const MONEY_PER_CONVERSATION_REP = 1.7 * 10000;

/** Formatea un monto COP con separador de miles en punto (es-CO), sin comas. */
function formatCop(value: number): string {
  const rounded = Math.round(value);
  return (
    "$" +
    rounded
      .toString()
      .replace(/\B(?=(\d{3})+(?!\d))/g, ".")
  );
}

export function RoiCalculator() {
  const [conversations, setConversations] = useState(100);
  const [reps, setReps] = useState(5);

  const hoursLost = Math.round(conversations * reps * 0.4);
  const moneyLost = Math.round(conversations * reps * MONEY_PER_CONVERSATION_REP);

  return (
    <section className="bg-background py-16 lg:py-24">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="rounded-2xl bg-primary p-8 lg:p-12 shadow-lg shadow-primary/10">
          <div className="grid lg:grid-cols-2 gap-10 items-center">
            {/* Controls */}
            <div className="space-y-8">
              <div className="space-y-3">
                <h2 className="font-heading text-3xl sm:text-4xl font-bold text-primary-foreground tracking-tight text-balance">
                  {copy("marketing", "roiTitle")}
                </h2>
                <p className="text-primary-foreground/80 text-lg leading-relaxed">
                  {copy("marketing", "roiLead")}
                </p>
              </div>
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label
                    htmlFor="roi-conversations"
                    className="text-primary-foreground/90 text-sm font-medium"
                  >
                    {copy("marketing", "roiConversationsLabel")}
                  </label>
                  <span className="text-primary-foreground font-bold text-sm tabular-nums">
                    {conversations}
                  </span>
                </div>
                <input
                  id="roi-conversations"
                  type="range"
                  min={50}
                  max={1000}
                  step={10}
                  value={conversations}
                  onChange={(e) => setConversations(Number(e.target.value))}
                  className="w-full accent-primary-foreground cursor-pointer"
                  aria-valuetext={`${conversations}`}
                />
              </div>
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label
                    htmlFor="roi-reps"
                    className="text-primary-foreground/90 text-sm font-medium"
                  >
                    {copy("marketing", "roiRepsLabel")}
                  </label>
                  <span className="text-primary-foreground font-bold text-sm tabular-nums">
                    {reps}
                  </span>
                </div>
                <input
                  id="roi-reps"
                  type="range"
                  min={1}
                  max={20}
                  step={1}
                  value={reps}
                  onChange={(e) => setReps(Number(e.target.value))}
                  className="w-full accent-primary-foreground cursor-pointer"
                  aria-valuetext={`${reps}`}
                />
              </div>
            </div>

            {/* Result card */}
            <div className="rounded-2xl bg-primary-foreground/10 backdrop-blur border border-primary-foreground/20 p-8">
              <div className="space-y-6">
                <div>
                  <div className="flex items-center gap-2 text-primary-foreground/70 text-sm mb-1">
                    <Clock className="w-4 h-4" />
                    <span>{copy("marketing", "roiHoursLost")}</span>
                  </div>
                  <p className="text-4xl font-bold text-primary-foreground tabular-nums">
                    {hoursLost}
                    <span className="text-xl font-semibold text-primary-foreground/80 ml-1">
                      h/sem
                    </span>
                  </p>
                </div>
                <div className="border-t border-primary-foreground/15 pt-6">
                  <div className="flex items-center gap-2 text-primary-foreground/70 text-sm mb-1">
                    <TrendingUp className="w-4 h-4" />
                    <span>{copy("marketing", "roiMoneyLost")}</span>
                  </div>
                  <p className="text-4xl font-bold text-primary-foreground tabular-nums">
                    {formatCop(moneyLost)}
                  </p>
                </div>
                <Link href="/signup" className="block">
                  <Button
                    size="lg"
                    className="w-full bg-white text-primary hover:bg-white/90 rounded-xl font-semibold"
                  >
                    {copy("marketing", "roiCta")}
                  </Button>
                </Link>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
