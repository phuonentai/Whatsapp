"use client";

import { useState } from "react";
import { TicketList } from "./ticket-list";
import { TicketDetail } from "./ticket-detail";

export function TicketsPanel() {
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [statusFilter, setStatusFilter] = useState("");

  return (
    <div className="grid grid-cols-1 lg:grid-cols-5 gap-4">
      <div className="lg:col-span-2">
        <TicketList
          selectedId={selectedId}
          onSelect={setSelectedId}
          statusFilter={statusFilter}
          onStatusFilterChange={setStatusFilter}
        />
      </div>
      <div className="lg:col-span-3">
        {selectedId ? (
          <TicketDetail id={selectedId} />
        ) : (
          <div className="text-gray-500 text-sm p-8 border rounded-lg bg-white">
            Selecciona un ticket para ver los detalles.
          </div>
        )}
      </div>
    </div>
  );
}
