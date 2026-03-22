"use client";

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

type Theme = "light" | "dark";
type ThemeSetting = Theme | "system";

type ThemeContextValue = {
  resolvedTheme: Theme;
  setTheme: (theme: ThemeSetting) => void;
};

type ThemeProviderProps = {
  children: ReactNode;
  defaultTheme?: ThemeSetting;
  enableSystem?: boolean;
  storageKey?: string;
};

const ThemeContext = createContext<ThemeContextValue | null>(null);

function getSystemTheme(): Theme {
  if (typeof window === "undefined") return "light";
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export function ThemeProvider({
  children,
  defaultTheme = "system",
  enableSystem = true,
  storageKey = "jobfinder-theme",
}: ThemeProviderProps) {
  const [resolvedTheme, setResolvedTheme] = useState<Theme>("light");

  useEffect(() => {
    const saved = localStorage.getItem(storageKey) as ThemeSetting | null;
    const initial = saved ?? defaultTheme;
    const next = initial === "system" ? (enableSystem ? getSystemTheme() : "light") : initial;
    setResolvedTheme(next);
  }, [defaultTheme, enableSystem, storageKey]);

  useEffect(() => {
    const root = document.documentElement;
    root.classList.remove("light", "dark");
    root.classList.add(resolvedTheme);
  }, [resolvedTheme]);

  const value = useMemo<ThemeContextValue>(
    () => ({
      resolvedTheme,
      setTheme: (theme) => {
        const next =
          theme === "system" ? (enableSystem ? getSystemTheme() : "light") : theme;
        localStorage.setItem(storageKey, theme);
        setResolvedTheme(next);
      },
    }),
    [enableSystem, resolvedTheme, storageKey],
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme() {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error("useTheme must be used within ThemeProvider");
  }
  return context;
}
