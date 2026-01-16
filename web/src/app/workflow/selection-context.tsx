import * as React from "react";
import { create } from "zustand";

interface SelectionStore {
  selectedTaskId: string;
  setSelectedTaskId: (id: string) => void;
}

export const useSelectionStore = create<SelectionStore>((set) => ({
  selectedTaskId: "",
  setSelectedTaskId: (id) => set({ selectedTaskId: id }),
}));

export function useSelection() {
  return useSelectionStore();
}

/**
 * @deprecated Provider is no longer needed with zustand store
 */
export function SelectionProvider({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

