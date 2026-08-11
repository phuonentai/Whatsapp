"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { Loader2, Search } from "lucide-react";

import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "@/components/ui/command";
import { DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { commandRegistry } from "@/lib/command-registry";
import { crmRepository } from "@/lib/api/api/repositories/crm-repository";
import type { ContactDto } from "@/lib/api/api/dto/crm.dto";
import {
  useCommandPaletteStore,
  type CommandPaletteMode,
} from "@/lib/stores/command-palette-store";

const SEARCH_DEBOUNCE_MS = 300;

function useDebouncedValue(value: string, delay: number): string {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const timer = window.setTimeout(() => setDebounced(value), delay);
    return () => window.clearTimeout(timer);
  }, [value, delay]);
  return debounced;
}

interface PaletteBodyProps {
  mode: CommandPaletteMode;
  onNavigate: (url: string) => void;
}

function PaletteBody({ mode, onNavigate }: PaletteBodyProps) {
  const [query, setQuery] = useState("");
  const [activeMode, setActiveMode] = useState<"command" | "search">(mode);
  const inputRef = useRef<HTMLInputElement>(null);
  const promotionTimerRef = useRef<number | null>(null);

  const debouncedQuery = useDebouncedValue(query, SEARCH_DEBOUNCE_MS);

  // Focus the palette input on open (external DOM update).
  useEffect(() => {
    const timer = window.setTimeout(() => inputRef.current?.focus(), 0);
    return () => window.clearTimeout(timer);
  }, []);

  // Clear a pending command→search promotion on unmount.
  useEffect(() => {
    return () => {
      if (promotionTimerRef.current !== null) {
        window.clearTimeout(promotionTimerRef.current);
      }
    };
  }, []);

  // Search results are delegated to the existing searchContacts route.
  const searchQuery = useQuery({
    queryKey: ["command-palette-search", debouncedQuery],
    queryFn: () => crmRepository.searchContacts(debouncedQuery, { limit: 20 }),
    enabled: activeMode === "search" && debouncedQuery.trim().length > 0,
    staleTime: 30_000,
  });

  const searchResults = searchQuery.data?.items ?? [];

  const filteredDestinations = useMemo(() => {
    if (activeMode !== "command") return [];
    const term = query.trim().toLowerCase();
    if (!term) return commandRegistry;
    return commandRegistry.filter((destination) => {
      const haystack = `${destination.title} ${destination.keywords?.join(" ") ?? ""}`.toLowerCase();
      return haystack.includes(term);
    });
  }, [activeMode, query]);

  const promoteToSearch = useCallback(() => {
    if (promotionTimerRef.current !== null) {
      window.clearTimeout(promotionTimerRef.current);
    }
    const trimmed = query.trim();
    if (trimmed.length >= 2 && filteredDestinations.length === 0) {
      promotionTimerRef.current = window.setTimeout(() => {
        promotionTimerRef.current = null;
        setActiveMode((current) => (current === "search" ? current : "search"));
      }, SEARCH_DEBOUNCE_MS);
    }
  }, [query, filteredDestinations.length]);

  const handleValueChange = (value: string) => {
    setQuery(value);
    if (activeMode === "command") {
      promoteToSearch();
    }
  };

  const sections = useMemo(() => {
    const map = new Map<string, typeof commandRegistry>();
    for (const destination of filteredDestinations) {
      const list = map.get(destination.section) ?? [];
      list.push(destination);
      map.set(destination.section, list);
    }
    return Array.from(map.entries());
  }, [filteredDestinations]);

  const isLoadingSearch = searchQuery.isFetching && searchQuery.data === undefined;

  return (
    <>
      <CommandInput
        ref={inputRef}
        value={query}
        onValueChange={handleValueChange}
        placeholder={
          activeMode === "search"
            ? "Search contacts…"
            : "Type a command or search…"
        }
      />
      <CommandList>
        {activeMode === "command" ? (
          <>
            <CommandEmpty>No commands found.</CommandEmpty>
            {sections.map(([section, destinations]) => (
              <CommandGroup key={section} heading={section}>
                {destinations.map((destination) => (
                  <CommandItem
                    key={destination.id}
                    value={`${destination.title} ${destination.keywords?.join(" ") ?? ""}`.toLowerCase()}
                    onSelect={() => onNavigate(destination.url)}
                  >
                    <destination.icon className="mr-2 h-4 w-4" aria-hidden />
                    <span>{destination.title}</span>
                  </CommandItem>
                ))}
              </CommandGroup>
            ))}
          </>
        ) : (
          <>
            {isLoadingSearch ? (
              <div className="flex items-center gap-2 px-4 py-4 text-sm text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
                Searching contacts…
              </div>
            ) : searchResults.length > 0 ? (
              <CommandGroup heading="Contacts">
                {searchResults.map((contact: ContactDto) => (
                  <CommandItem
                    key={contact.id}
                    value={`${contact.display_name} ${contact.phone_number}`.toLowerCase()}
                    onSelect={() =>
                      onNavigate(`/dashboard/crm?view=contactos&id=${contact.id}`)
                    }
                  >
                    <Search className="mr-2 h-4 w-4" aria-hidden />
                    <span>{contact.display_name || contact.phone_number}</span>
                    {contact.phone_number ? (
                      <span className="ml-2 text-xs text-muted-foreground">
                        {contact.phone_number}
                      </span>
                    ) : null}
                  </CommandItem>
                ))}
              </CommandGroup>
            ) : (
              <CommandEmpty>
                {debouncedQuery.trim().length === 0
                  ? "Type a name or phone to search contacts."
                  : "No results"}
              </CommandEmpty>
            )}
          </>
        )}
      </CommandList>
      <CommandSeparator />
      <div className="flex items-center justify-between px-3 py-2 text-xs text-muted-foreground">
        <span>↑↓ to navigate</span>
        <span>↵ to open</span>
        <span>esc to close</span>
      </div>
    </>
  );
}

export function CommandPalette() {
  const router = useRouter();
  const { open, mode, session, closePalette } = useCommandPaletteStore();

  const navigate = useCallback(
    (url: string) => {
      closePalette();
      router.push(url);
    },
    [closePalette, router]
  );

  return (
    <CommandDialog open={open} onOpenChange={(next) => !next && closePalette()}>
      <DialogTitle className="sr-only">Command palette</DialogTitle>
      <DialogDescription className="sr-only">
        Search across the app and navigate with the keyboard.
      </DialogDescription>
      {open ? <PaletteBody key={session} mode={mode} onNavigate={navigate} /> : null}
    </CommandDialog>
  );
}
