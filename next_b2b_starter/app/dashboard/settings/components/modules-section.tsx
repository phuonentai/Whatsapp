"use client";

import { useState } from "react";
import { useModulesCatalogQuery, useOrgModulesQuery } from "@/lib/hooks/queries/use-modules-queries";
import { useSaveModuleConfig } from "@/lib/hooks/mutations/use-tickets-mutations";
import { useModule } from "@/lib/hooks/use-entitlement";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";

export function ModulesSection() {
  const { data: catalog, isLoading } = useModulesCatalogQuery();
  const { data: orgModules } = useOrgModulesQuery();

  if (isLoading) {
    return <div className="text-sm text-gray-500">Cargando módulos...</div>;
  }

  return (
    <div className="space-y-4">
      {(catalog ?? []).map((module) => (
        <ModuleCard key={module.key} module={module} orgConfig={orgModules?.find((m) => m.key === module.key)?.config} />
      ))}
    </div>
  );
}

function ModuleCard({
  module,
  orgConfig,
}: {
  module: { key: string; name: string; description?: string };
  orgConfig?: Record<string, unknown>;
}) {
  const { enabled } = useModule(module.key);
  const saveConfig = useSaveModuleConfig();

  const [slaHours, setSlaHours] = useState<string>("");
  const [priorities, setPriorities] = useState<string>("");
  const [tags, setTags] = useState<string>("");

  // Sanctioned render-phase state adjustment (React "adjusting state during
  // render" pattern): re-seed the form when orgConfig changes, no effect needed.
  const [prevConfig, setPrevConfig] = useState(orgConfig);
  if (orgConfig !== prevConfig) {
    setPrevConfig(orgConfig);
    const config = orgConfig as
      | { sla_hours?: Record<string, number>; priorities?: string[]; tags?: string[] }
      | undefined;
    setSlaHours(
      config?.sla_hours ? Object.entries(config.sla_hours).map(([k, v]) => `${k}:${v}`).join(",") : ""
    );
    setPriorities(config?.priorities?.join(",") ?? "");
    setTags(config?.tags?.join(",") ?? "");
  }

  const handleSave = () => {
    const config: Record<string, unknown> = {};
    if (slaHours.trim()) {
      const sla: Record<string, number> = {};
      for (const pair of slaHours.split(",")) {
        const [k, v] = pair.split(":").map((s) => s.trim());
        if (k && v) sla[k] = Number(v);
      }
      if (Object.keys(sla).length) config.sla_hours = sla;
    }
    if (priorities.trim()) config.priorities = priorities.split(",").map((s) => s.trim());
    if (tags.trim()) config.tags = tags.split(",").map((s) => s.trim());
    saveConfig.mutate({ key: module.key, config });
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">{module.name}</CardTitle>
          <Badge variant={enabled ? "default" : "outline"}>{enabled ? "Activo" : "No adquirido"}</Badge>
        </div>
        <CardDescription>{module.description}</CardDescription>
      </CardHeader>
      {enabled && (
        <CardContent className="space-y-3">
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <div>
              <Label className="text-xs">SLA horas (prioridad:horas)</Label>
              <Input
                value={slaHours}
                onChange={(e) => setSlaHours(e.target.value)}
                placeholder="high:8,normal:24,low:48"
              />
            </div>
            <div>
              <Label className="text-xs">Prioridades</Label>
              <Input
                value={priorities}
                onChange={(e) => setPriorities(e.target.value)}
                placeholder="low,normal,high"
              />
            </div>
            <div>
              <Label className="text-xs">Tags</Label>
              <Input
                value={tags}
                onChange={(e) => setTags(e.target.value)}
                placeholder="billing,feature,bug"
              />
            </div>
          </div>
          <Button onClick={handleSave} disabled={saveConfig.isPending} className="bg-gray-900 text-white hover:bg-gray-800">
            Guardar configuración
          </Button>
        </CardContent>
      )}
    </Card>
  );
}
