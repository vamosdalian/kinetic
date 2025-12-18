import * as React from "react";

interface SelectionContextType {
  selectedTaskId: string;
  setSelectedTaskId: (id: string) => void;
}

const SelectionContext = React.createContext<SelectionContextType | null>(null);

export function SelectionProvider({ children }: { children: React.ReactNode }) {
  const [selectedTaskId, setSelectedTaskId] = React.useState("");

  return (
    <SelectionContext.Provider value={{ selectedTaskId, setSelectedTaskId }}>
      {children}
    </SelectionContext.Provider>
  );
}

export function useSelection() {
  const context = React.useContext(SelectionContext);
  if (!context) {
    throw new Error("useSelection must be used within a SelectionProvider");
  }
  return context;
}

