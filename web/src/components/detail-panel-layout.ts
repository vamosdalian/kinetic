import type { CSSProperties } from "react";

export const DETAIL_PANEL_WIDTH_PX = 500;
const DETAIL_PANEL_BUTTON_SHIFT_PX = DETAIL_PANEL_WIDTH_PX + 10;

export const DETAIL_PANEL_LAYOUT_STYLE = {
  "--detail-panel-width": `${DETAIL_PANEL_WIDTH_PX}px`,
  "--detail-panel-button-shift": `${DETAIL_PANEL_BUTTON_SHIFT_PX}px`,
} as CSSProperties;
