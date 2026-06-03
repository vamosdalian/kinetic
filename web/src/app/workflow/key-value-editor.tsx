import { useEffect, useRef, useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

interface KeyValueEditorProps {
  values: Record<string, string>;
  onChange: (values: Record<string, string>) => void;
  emptyText?: string;
  keyPlaceholder: string;
  valuePlaceholder: string;
  keyPrefix?: string;
  showAddButton?: boolean;
}

interface Row {
  id: number;
  key: string;
  value: string;
}

function buildNextMapKey(usedKeys: Set<string>, prefix: string) {
  let index = usedKeys.size + 1;
  let candidate = `${prefix}-${index}`;

  while (usedKeys.has(candidate)) {
    index += 1;
    candidate = `${prefix}-${index}`;
  }

  return candidate;
}

// Collapse the editable rows back into a plain map. Rows with empty keys are
// skipped, and duplicate keys keep the last value wins so the parent only ever
// sees a valid Record.
function rowsToValues(rows: Row[]): Record<string, string> {
  const result: Record<string, string> = {};
  for (const row of rows) {
    if (row.key.length === 0) continue;
    result[row.key] = row.value;
  }
  return result;
}

export function KeyValueEditor({
  values,
  onChange,
  emptyText,
  keyPlaceholder,
  valuePlaceholder,
  keyPrefix = "item",
  showAddButton = true,
}: KeyValueEditorProps) {
  // Keep an internal row list with stable ids so editing a key never remounts
  // its input (which would drop focus) and never reorders the list.
  const nextId = useRef(0);
  const [rows, setRows] = useState<Row[]>(() =>
    Object.entries(values).map(([key, value]) => ({ id: nextId.current++, key, value })),
  );

  // Re-sync from props only when the external map differs from what our rows
  // currently represent (e.g. the form was reset or loaded fresh data).
  useEffect(() => {
    const current = rowsToValues(rows);
    const sameLength = Object.keys(current).length === Object.keys(values).length;
    const sameEntries =
      sameLength && Object.entries(values).every(([key, value]) => current[key] === value);
    if (sameEntries) return;

    setRows(Object.entries(values).map(([key, value]) => ({ id: nextId.current++, key, value })));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [values]);

  const commit = (nextRows: Row[]) => {
    setRows(nextRows);
    onChange(rowsToValues(nextRows));
  };

  return (
    <div className="grid gap-3">
      {showAddButton && (
        <div className="flex items-center justify-end gap-3">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => {
              const usedKeys = new Set(rows.map((row) => row.key));
              commit([
                ...rows,
                { id: nextId.current++, key: buildNextMapKey(usedKeys, keyPrefix), value: "" },
              ]);
            }}
          >
            Add
          </Button>
        </div>
      )}

      {rows.length > 0 ? (
        <div className="grid gap-3">
          {rows.map((row) => (
            <div key={row.id} className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] gap-2">
              <Input
                placeholder={keyPlaceholder}
                value={row.key}
                onChange={(e) => {
                  const nextKey = e.target.value;
                  commit(rows.map((r) => (r.id === row.id ? { ...r, key: nextKey } : r)));
                }}
              />
              <Input
                placeholder={valuePlaceholder}
                value={row.value}
                onChange={(e) => {
                  const nextValue = e.target.value;
                  commit(rows.map((r) => (r.id === row.id ? { ...r, value: nextValue } : r)));
                }}
              />
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  commit(rows.filter((r) => r.id !== row.id));
                }}
              >
                Remove
              </Button>
            </div>
          ))}
        </div>
      ) : emptyText ? (
        <p className="text-xs text-muted-foreground">{emptyText}</p>
      ) : null}
    </div>
  );
}
