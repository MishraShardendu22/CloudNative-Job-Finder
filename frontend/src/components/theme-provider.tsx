"use client";

import {
  createContext,
  type ReactNode,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";

type Theme = "dark";
type ThemeSetting = "dark";

type ThemeContextValue = {
  resolvedTheme: Theme;
  setTheme: (theme: ThemeSetting) => void;
};

type ThemeProviderProps = {
  children: ReactNode;
  defaultTheme?: ThemeSetting;
  storageKey?: string;
};

const ThemeContext = createContext<ThemeContextValue | null>(null);

export function ThemeProvider({
  children,
  defaultTheme = "dark",
  storageKey = "jobfinder-theme",
}: ThemeProviderProps) {
  const [resolvedTheme, setResolvedTheme] = useState<Theme>("dark");

  useEffect(() => {
    localStorage.setItem(storageKey, defaultTheme);
    setResolvedTheme("dark");
  }, [defaultTheme, storageKey]);

  useEffect(() => {
    const root = document.documentElement;
    root.classList.remove("light", "dark");
    root.classList.add(resolvedTheme);
  }, [resolvedTheme]);

  const value = useMemo<ThemeContextValue>(
    () => ({
      resolvedTheme,
      setTheme: (theme) => {
        if (theme !== "dark") return;
        localStorage.setItem(storageKey, "dark");
        setResolvedTheme("dark");
      },
    }),
    [resolvedTheme, storageKey],
  );

  return (
    <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
  );
}

export function useTheme() {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error("useTheme must be used within ThemeProvider");
  }
  return context;
}
