"use client";

import { useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { useQueryClient } from "@tanstack/react-query";
import { useVirtualizer } from "@tanstack/react-virtual";
import { ArrowDown, ArrowUp, ChevronLeft, ChevronRight, Download, Upload } from "lucide-react";
import { useContactsQuery } from "@/lib/hooks/queries/use-crm-queries";
import { queryKeys } from "@/lib/hooks/queries/query-keys";
import { useDeleteContact } from "@/lib/hooks/mutations/use-crm-mutations";
import { useFeature } from "@/lib/hooks/use-entitlement";
import { usePermissions } from "@/lib/hooks/use-permissions";
import type { ContactDto } from "@/lib/api/api/dto/crm.dto";
import { crmRepository } from "@/lib/api/api/repositories/crm-repository";
import { ContactDialog } from "@/components/crm/contact-dialog";
import { ConfirmDialog } from "@/components/crm/confirm-dialog";
import { ContactImportDialog } from "@/components/crm/contact-import-dialog";
import { ErrorState } from "@/components/common/error-state";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useCsvExport } from "@/lib/csv-export";

const PAGE_SIZE = 25;
const VIRTUALIZE_THRESHOLD = 100;
const ROW_HEIGHT = 52;

type SortKey = "display_name" | "phone_number" | "email" | "last_message_at" | "created_at";

const SORTABLE_COLUMNS: { key: SortKey; label: string }[] = [
  { key: "display_name", label: "Nombre" },
  { key: "phone_number", label: "Teléfono" },
  { key: "email", label: "Correo" },
  { key: "last_message_at", label: "Último Contacto" },
];

export function ContactTable() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [search, setSearch] = useState("");
  const [filterStatus, setFilterStatus] = useState("");
  const [dialogOpen, setDialogOpen] = useState(false);
  const [importOpen, setImportOpen] = useState(false);
  const [editing, setEditing] = useState<ContactDto | null>(null);
  const [deleting, setDeleting] = useState<ContactDto | null>(null);
  const [page, setPage] = useState(1);
  const [sortKey, setSortKey] = useState<SortKey>("created_at");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("desc"); // default: newest first
  const [selected, setSelected] = useState<Set<number>>(new Set());
  const [bulkDeleteOpen, setBulkDeleteOpen] = useState(false);
  const [isBulkDeleting, setIsBulkDeleting] = useState(false);
  const tableRef = useRef<HTMLDivElement>(null);
  const scrollRef = useRef<HTMLDivElement>(null);
  const canManage = useFeature("crm_contacts_manage");
  const { hasPermission } = usePermissions();
  const canExport = hasPermission("contact:export");
  const { data: contacts, isLoading, isError, refetch, isRefetching, total } = useContactsQuery({
    lead_status: filterStatus || undefined,
    page,
    pageSize: PAGE_SIZE,
  });
  const deleteMutation = useDeleteContact();
  const { isExporting, handleExport } = useCsvExport({
    run: () => crmRepository.exportContacts(),
    successMessage: "Contactos exportados",
    errorMessage: "Error al exportar contactos",
  });

  const totalPages = Math.max(1, Math.ceil((total ?? 0) / PAGE_SIZE));

  const goToPage = (next: number) => {
    const clamped = Math.min(Math.max(1, next), totalPages);
    if (clamped === page) return;
    setPage(clamped);
    tableRef.current?.scrollIntoView({ block: "start", behavior: "smooth" });
  };

  const handleSearchChange = (value: string) => {
    setSearch(value);
    setPage(1);
  };

  const handleFilterChange = (value: string) => {
    setFilterStatus(value);
    setPage(1);
  };

  const clearFilters = () => {
    setSearch("");
    setFilterStatus("");
    setPage(1);
  };

  const filtered = useMemo(() => {
    const query = search.trim().toLowerCase();
    if (!query) return contacts;
    return contacts?.filter((c) =>
      [c.display_name, c.phone_number, c.email]
        .filter(Boolean)
        .some((value) => (value as string).toLowerCase().includes(query))
    );
  }, [contacts, search]);

  const sorted = useMemo(() => {
    if (!filtered) return filtered;
    const dir = sortDir === "asc" ? 1 : -1;
    return [...filtered].sort((a, b) => {
      const av = a[sortKey];
      const bv = b[sortKey];
      if (av == null && bv == null) return 0;
      if (av == null) return 1;
      if (bv == null) return -1;
      return String(av).localeCompare(String(bv), "es") * dir;
    });
  }, [filtered, sortKey, sortDir]);

  const toggleSort = (key: SortKey) => {
    if (key === sortKey) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(key);
      setSortDir("asc");
    }
  };

  const rows = useMemo(() => sorted ?? [], [sorted]);
  const virtualize = rows.length > VIRTUALIZE_THRESHOLD;
  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => ROW_HEIGHT,
    overscan: 10,
  });
  const virtualItems = virtualize
    ? virtualizer.getVirtualItems()
    : rows.map((_, index) => ({ index, key: index, start: index * ROW_HEIGHT, size: ROW_HEIGHT }));

  const pageIds = useMemo(() => rows.map((c) => c.id), [rows]);
  const allSelected = rows.length > 0 && pageIds.every((id) => selected.has(id));
  const someSelected = pageIds.some((id) => selected.has(id));

  const toggleRow = (id: number) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleAll = () => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (allSelected) pageIds.forEach((id) => next.delete(id));
      else pageIds.forEach((id) => next.add(id));
      return next;
    });
  };

  const handleDelete = async () => {
    if (!deleting) return;
    try {
      await deleteMutation.mutateAsync(deleting.id);
      toast.success("Contacto eliminado");
      setDeleting(null);
    } catch {
      // error toast handled by mutation
    }
  };

  const handleBulkDelete = async () => {
    if (selected.size === 0) return;
    setIsBulkDeleting(true);
    let ok = 0;
    let failed = 0;
    // Sequential per-item deletes (org-scoped, idempotent); aggregate result.
    for (const id of selected) {
      try {
        await crmRepository.deleteContact(id);
        ok += 1;
      } catch {
        failed += 1;
      }
    }
    setIsBulkDeleting(false);
    setBulkDeleteOpen(false);
    setSelected(new Set());
    queryClient.invalidateQueries({ queryKey: queryKeys.crm.all });
    const parts: string[] = [];
    if (ok > 0) parts.push(`${ok} eliminados`);
    if (failed > 0) parts.push(`${failed} fallaron`);
    toast.success(parts.join(", "));
  };

  const openDetail = (c: ContactDto) => {
    router.push(`/dashboard/crm?view=contactos&id=${c.id}`);
  };

  const handleRowKeyDown = (e: React.KeyboardEvent<HTMLTableRowElement>, c: ContactDto) => {
    if (e.target !== e.currentTarget) return;
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      openDetail(c);
    }
  };

  if (isLoading) {
    return (
      <div>
        <div className="flex gap-4 mb-4 items-center">
          <Skeleton className="h-10 w-64" />
          <Skeleton className="h-10 w-40" />
          <Skeleton className="h-10 w-32" />
        </div>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-10" />
              <TableHead>Nombre</TableHead>
              <TableHead>Teléfono</TableHead>
              <TableHead>Correo</TableHead>
              <TableHead>Documento</TableHead>
              <TableHead>Empresa</TableHead>
              <TableHead>Estado</TableHead>
              <TableHead>Último Contacto</TableHead>
              <TableHead className="w-40">Acciones</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {Array.from({ length: 5 }).map((_, i) => (
              <TableRow key={i}>
                <TableCell><Skeleton className="h-4 w-4" /></TableCell>
                <TableCell><Skeleton className="h-4 w-40" /></TableCell>
                <TableCell><Skeleton className="h-4 w-32" /></TableCell>
                <TableCell><Skeleton className="h-4 w-44" /></TableCell>
                <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                <TableCell><Skeleton className="h-5 w-16" /></TableCell>
                <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                <TableCell><Skeleton className="h-4 w-16" /></TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    );
  }

  if (isError) {
    return (
      <ErrorState
        title="Error al cargar los contactos"
        description="No se pudieron cargar los contactos. Inténtalo de nuevo."
        onRetry={() => refetch()}
        isRetrying={isRefetching}
      />
    );
  }

  return (
    <div>
      <div className="flex gap-4 mb-4 items-center">
        <input
          type="text"
          placeholder="Buscar contactos..."
          value={search}
          onChange={(e) => handleSearchChange(e.target.value)}
          className="border rounded px-3 py-2 w-64"
        />
        <select
          value={filterStatus}
          onChange={(e) => handleFilterChange(e.target.value)}
          className="border rounded px-3 py-2"
        >
          <option value="">Todos los estados</option>
          <option value="nuevo">Nuevo</option>
          <option value="contactado">Contactado</option>
          <option value="calificado">Calificado</option>
          <option value="cliente">Cliente</option>
        </select>
        {canManage && (
          <Button
            onClick={() => {
              setEditing(null);
              setDialogOpen(true);
            }}
          >
            Nuevo contacto
          </Button>
        )}
        {canExport && (
          <Button variant="outline" onClick={handleExport} disabled={isExporting}>
            <Download className="mr-2 h-4 w-4" />
            {isExporting ? "Exportando..." : "Exportar"}
          </Button>
        )}
        {canManage && (
          <Button variant="outline" onClick={() => setImportOpen(true)}>
            <Upload className="mr-2 h-4 w-4" />
            Importar
          </Button>
        )}
      </div>

      {selected.size > 0 && (
        <div className="flex items-center justify-between gap-4 mb-4 px-4 py-2 bg-gray-100 border rounded-lg">
          <span className="text-sm font-medium">
            {selected.size} {selected.size === 1 ? "contacto seleccionado" : "contactos seleccionados"}
          </span>
          <div className="flex items-center gap-2">
            {canExport && (
              <Button
                variant="outline"
                size="sm"
                aria-label="Exportar seleccionados"
                onClick={handleExport}
                disabled={isExporting}
              >
                <Download className="mr-2 h-4 w-4" />
                Exportar
              </Button>
            )}
            {canManage && (
              <Button
                variant="destructive"
                size="sm"
                aria-label="Eliminar seleccionados"
                onClick={() => setBulkDeleteOpen(true)}
                disabled={isBulkDeleting}
              >
                Eliminar
              </Button>
            )}
            <button
              onClick={() => setSelected(new Set())}
              className="text-sm text-gray-500 hover:underline"
            >
              Limpiar
            </button>
          </div>
        </div>
      )}

      <div ref={tableRef}>
        <div ref={scrollRef} className={virtualize ? "max-h-[70vh] overflow-y-auto" : ""}>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-10">
                  <Checkbox
                    aria-label="Seleccionar todos"
                    checked={allSelected ? true : someSelected ? "indeterminate" : false}
                    onClick={(e) => e.stopPropagation()}
                    onCheckedChange={toggleAll}
                  />
                </TableHead>
                {SORTABLE_COLUMNS.map((col) => {
                  const active = sortKey === col.key;
                  return (
                    <TableHead
                      key={col.key}
                      aria-sort={active ? (sortDir === "asc" ? "ascending" : "descending") : undefined}
                      className="cursor-pointer select-none"
                      onClick={() => toggleSort(col.key)}
                    >
                      <span className="inline-flex items-center gap-1">
                        {col.label}
                        {active && (sortDir === "asc" ? <ArrowUp className="h-3 w-3" /> : <ArrowDown className="h-3 w-3" />)}
                      </span>
                    </TableHead>
                  );
                })}
                <TableHead>Documento</TableHead>
                <TableHead>Empresa</TableHead>
                <TableHead>Estado</TableHead>
                <TableHead className="w-40">Acciones</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody style={virtualize ? { height: virtualizer.getTotalSize(), position: "relative" } : undefined}>
              {virtualItems.map((vi) => {
                const c = rows[vi.index];
                return (
                  <TableRow
                    key={c.id}
                    tabIndex={0}
                    className="cursor-pointer"
                    data-state={selected.has(c.id) ? "selected" : undefined}
                    style={virtualize ? {
                      position: "absolute",
                      top: 0,
                      left: 0,
                      width: "100%",
                      height: vi.size,
                      transform: `translateY(${vi.start}px)`,
                    } : undefined}
                    onClick={() => openDetail(c)}
                    onKeyDown={(e) => handleRowKeyDown(e, c)}
                  >
                    <TableCell>
                      <Checkbox
                        aria-label={`Seleccionar ${c.display_name || c.phone_number}`}
                        checked={selected.has(c.id)}
                        onClick={(e) => e.stopPropagation()}
                        onCheckedChange={() => toggleRow(c.id)}
                      />
                    </TableCell>
                    <TableCell className="font-medium text-blue-600 hover:underline">{c.display_name || c.phone_number}</TableCell>
                    <TableCell>{c.phone_number}</TableCell>
                    <TableCell>{c.email || "-"}</TableCell>
                    <TableCell>{c.numero_documento ? `${c.tipo_documento} ${c.numero_documento}` : "-"}</TableCell>
                    <TableCell>{c.company_id ? `Empresa #${c.company_id}` : "-"}</TableCell>
                    <TableCell>
                      <span className={`px-2 py-1 rounded text-xs ${
                        c.lead_status === "cliente" ? "bg-green-100 text-green-800" :
                        c.lead_status === "calificado" ? "bg-blue-100 text-blue-800" :
                        c.lead_status === "nuevo" ? "bg-gray-100 text-gray-800" :
                        "bg-yellow-100 text-yellow-800"
                      }`}>
                        {c.lead_status}
                      </span>
                    </TableCell>
                    <TableCell className="text-sm text-gray-500">
                      {c.last_message_at ? new Date(c.last_message_at).toLocaleDateString("es-CO") : "-"}
                    </TableCell>
                    <TableCell>
                      {canManage && (
                        <div className="flex gap-2">
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              setEditing(c);
                              setDialogOpen(true);
                            }}
                            className="text-blue-600 hover:underline text-sm"
                          >
                            Editar
                          </button>
                          <button
                            aria-label="Eliminar"
                            onClick={(e) => {
                              e.stopPropagation();
                              setDeleting(c);
                            }}
                            className="text-red-600 hover:underline text-sm"
                          >
                            Eliminar
                          </button>
                        </div>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      </div>

      {rows.length === 0 && (
        contacts && contacts.length > 0 ? (
          <div className="p-4 text-center text-gray-400">
            No hay resultados para la búsqueda
            <button onClick={clearFilters} className="ml-2 text-blue-600 hover:underline">
              Limpiar filtros
            </button>
          </div>
        ) : (
          <div className="p-4 text-center text-gray-400">No hay contactos</div>
        )
      )}

      {totalPages > 1 && (
        <div className="flex items-center justify-between mt-4">
          <span className="text-sm text-gray-500">
            {total} {total === 1 ? "contacto" : "contactos"} · Página {page} de {totalPages}
          </span>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => goToPage(page - 1)} disabled={page <= 1}>
              <ChevronLeft className="h-4 w-4" />
              Anterior
            </Button>
            <Button variant="outline" size="sm" onClick={() => goToPage(page + 1)} disabled={page >= totalPages}>
              Siguiente
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}

      <ContactDialog open={dialogOpen} onOpenChange={setDialogOpen} contact={editing} />
      <ContactImportDialog open={importOpen} onOpenChange={setImportOpen} />
      <ConfirmDialog
        open={Boolean(deleting)}
        onOpenChange={(next) => !next && setDeleting(null)}
        title="Eliminar contacto"
        description={`¿Estás seguro de eliminar a ${deleting?.display_name || deleting?.phone_number}? Esta acción no se puede deshacer.`}
        confirmLabel="Eliminar"
        loading={deleteMutation.isPending}
        onConfirm={handleDelete}
      />
      <ConfirmDialog
        open={bulkDeleteOpen}
        onOpenChange={setBulkDeleteOpen}
        title={`Eliminar ${selected.size} contactos`}
        description="Esta acción no se puede deshacer."
        confirmLabel="Eliminar"
        loading={isBulkDeleting}
        onConfirm={handleBulkDelete}
      />
    </div>
  );
}
