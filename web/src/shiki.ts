import { createHighlighterCore, type HighlighterCore } from "shiki/core";
import { createJavaScriptRegexEngine } from "shiki/engine/javascript";
import json from "shiki/langs/json.mjs";
import githubDark from "shiki/themes/github-dark.mjs";
import githubLight from "shiki/themes/github-light.mjs";

let highlighter: Promise<HighlighterCore> | null = null;

export function getHighlighter(): Promise<HighlighterCore> {
  if (!highlighter) {
    highlighter = createHighlighterCore({
      themes: [githubDark, githubLight],
      langs: [json],
      engine: createJavaScriptRegexEngine(),
    });
  }
  return highlighter;
}

export const themeNameFor = (isDark: boolean) =>
  isDark ? "github-dark" : "github-light";
