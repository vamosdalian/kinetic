import { create } from "zustand";

interface DirtyState {
  isDirty: boolean;
  setDirty: (dirty: boolean) => void;
  markDirty: () => void;
  markClean: () => void;
}

export const useDirtyStore = create<DirtyState>()((set) => ({
  isDirty: false,
  setDirty: (dirty: boolean) => set({ isDirty: dirty }),
  markDirty: () => set({ isDirty: true }),
  markClean: () => set({ isDirty: false }),
}));

