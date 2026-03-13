# LEARNINGS

Accumulated lessons from debugging and development. Check here before touching CSS, layout, or build config.

---

## Tailwind v4 — unlayered CSS resets kill all utilities

**Symptom**: Tailwind classes like `px-6`, `py-8`, `gap-4` have zero effect. `getComputedStyle(el).paddingLeft` returns `"0px"` even though the class is in the stylesheet and computes correctly in isolation.

**Root cause**: An old `* { margin: 0; padding: 0; box-sizing: border-box; }` block existed in `index.css` *outside* any `@layer`. In CSS cascade layers, unlayered rules have higher priority than ALL named layers. Tailwind places every utility in `@layer utilities`, so the bare `*` reset silently wins everywhere.

The tricky part: `<header>` elements happened to look fine because a *different* unlayered rule `header { padding: 20px }` overrode the `*` reset with higher specificity. This made debugging confusing — some elements had padding, others didn't.

**Fix**: Remove the unlayered reset. Tailwind's own `@layer base` preflight already provides `* { margin: 0; padding: 0 }` at the correct cascade priority. If any legacy CSS must coexist, wrap it in `@layer base {}` so utilities can override it.

**Rule going forward**: `index.css` must not contain *any* bare CSS rules outside a `@layer`. Structure:
```css
@import "tailwindcss";
/* @theme inline { ... } is OK — it's not a rule layer */

:root { /* design tokens only, no rules */ }
.dark { /* design tokens only, no rules */ }

@layer base {
  /* element-level resets and legacy CSS go here */
}
/* No bare rules below this point */
```

---

## Nested `<main>` inside `<SidebarInset>`

`SidebarInset` from shadcn/ui renders as a `<main>` element. Wrapping its content in another `<main className="flex flex-1 flex-col overflow-auto">` creates a nested landmark that confuses both accessibility and the flex layout chain.

**Fix**: Use a plain `<div>` as the content wrapper:
```jsx
<SidebarInset>
  <AppHeader ... />
  <div className="flex flex-1 flex-col gap-4 px-4 py-6">
    {/* page content */}
  </div>
</SidebarInset>
```

---

## shadcn-admin spacing reference values

When matching the spacious felt of the shadcn-admin reference app:

| Element | Class |
|---------|-------|
| Main content padding | `px-4 py-6` |
| Grid gap (stat cards) | `gap-4` |
| Section spacing within a page | `space-y-4` |
| CardHeader on stat cards | `className="flex flex-row items-center justify-between space-y-0 pb-2"` |
| `pb-2` on stat card CardHeader | **intentional** — keeps label compact, do not remove |

Spaciousness in shadcn-admin comes from clean component structure, not inflated padding values.
