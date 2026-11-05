export const THEME_STORAGE_KEY = "atb-ui-store-v1";
export const THEME_ATTRIBUTE = "data-theme";

export type ThemePreference = "system" | "light" | "dark";

export function isThemePreference(value: unknown): value is ThemePreference {
  return value === "system" || value === "light" || value === "dark";
}

export function resolveTheme(preference: ThemePreference, systemPrefersDark: boolean): "light" | "dark" {
  if (preference === "system") return systemPrefersDark ? "dark" : "light";
  return preference;
}

export const themeBootstrapScript = `(function(){try{var raw=localStorage.getItem("${THEME_STORAGE_KEY}");var preference="system";if(raw){var persisted=JSON.parse(raw);if(persisted&&persisted.state&&(persisted.state.theme==="light"||persisted.state.theme==="dark"||persisted.state.theme==="system")){preference=persisted.state.theme;}}var dark=preference==="dark"||(preference==="system"&&window.matchMedia("(prefers-color-scheme: dark)").matches);document.documentElement.dataset.theme=dark?"dark":"light";}catch(e){}})();`;
