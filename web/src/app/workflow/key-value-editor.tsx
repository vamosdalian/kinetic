import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

interface KeyValueEditorProps {
  values: Record<string, string>;
  onChange: (values: Record<string, string>) => void;
  emptyText: string;
  keyPlaceholder: string;
  valuePlaceholder: string;
  keyPrefix?: string;
  showAddButton?: boolean;
}

function buildNextMapKey(values: Record<string, string>, prefix: string) {
  let index = Object.keys(values).length + 1;
  let candidate = `${prefix}-${index}`;

  while (candidate in values) {
    index += 1;
    candidate = `${prefix}-${index}`;
  }

  return candidate;
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
  const entries = Object.entries(values);

  return (
    <div className="grid gap-3">
      {showAddButton && (
        <div className="flex items-center justify-end gap-3">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => {
              const nextValues = { ...values };
              nextValues[buildNextMapKey(nextValues, keyPrefix)] = "";
              onChange(nextValues);
            }}
          >
            Add
          </Button>
        </div>
      )}

      {entries.length > 0 ? (
        <div className="grid gap-3">
          {entries.map(([key, value]) => (
            <div key={key} className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] gap-2">
              <Input
                placeholder={keyPlaceholder}
                value={key}
                onChange={(e) => {
                  const nextKey = e.target.value;
                  const nextValues = { ...values };
                  delete nextValues[key];
                  nextValues[nextKey] = value;
                  onChange(nextValues);
                }}
              />
              <Input
                placeholder={valuePlaceholder}
                value={value}
                onChange={(e) => {
                  onChange({
                    ...values,
                    [key]: e.target.value,
                  });
                }}
              />
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  const nextValues = { ...values };
                  delete nextValues[key];
                  onChange(nextValues);
                }}
              >
                Remove
              </Button>
            </div>
          ))}
        </div>
      ) : (
        <p className="text-xs text-muted-foreground">{emptyText}</p>
      )}
    </div>
  );
}