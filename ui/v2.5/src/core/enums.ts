type ImageLightboxDisplayMode = "ORIGINAL" | "FIT_XY" | "FIT_X";
type ImageLightboxScrollMode = "ZOOM" | "PAN_Y";

export const imageLightboxDisplayModeIntlMap = new Map<
  ImageLightboxDisplayMode,
  string
>([
  [
    "ORIGINAL" as ImageLightboxDisplayMode,
    "dialogs.lightbox.display_mode.original",
  ],
  [
    "FIT_XY" as ImageLightboxDisplayMode,
    "dialogs.lightbox.display_mode.fit_to_screen",
  ],
  [
    "FIT_X" as ImageLightboxDisplayMode,
    "dialogs.lightbox.display_mode.fit_horizontally",
  ],
]);

export const imageLightboxScrollModeIntlMap = new Map<
  ImageLightboxScrollMode,
  string
>([
  ["ZOOM" as ImageLightboxScrollMode, "dialogs.lightbox.scroll_mode.zoom"],
  ["PAN_Y" as ImageLightboxScrollMode, "dialogs.lightbox.scroll_mode.pan_y"],
]);
