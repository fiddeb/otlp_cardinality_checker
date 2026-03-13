# Proposal: Migrate to ShadcnUI Design System

**Change ID:** `migrate-shadcn-ui`  
**Status:** Draft  
**Created:** 2026-01-22  
**Author:** System

## Overview

Gradually migrate the OTLP Cardinality Checker web UI from custom CSS to the shadcn-admin design system, preserving all existing functionality while improving maintainability, consistency, and user experience.

## Motivation

The current web UI uses custom CSS and basic React components. While functional, it lacks:

1. **Design consistency** - Ad-hoc styling leads to inconsistent spacing, colors, and interactions
2. **Accessibility** - Missing ARIA labels, keyboard navigation, and screen reader support
3. **Maintainability** - CSS growing complex with dark mode duplications
4. **Modern UX patterns** - Missing components like sidebars, drawers, proper tables
5. **Mobile responsiveness** - Limited mobile optimization

The [shadcn-admin](https://github.com/satnaing/shadcn-admin) project provides:
- Professional admin dashboard components (ShadcnUI + RadixUI)
- Built-in dark mode with theme switching
- TypeScript support
- Accessible components (WCAG compliant)
- Responsive design patterns
- Modern build tooling (Vite + Tailwind CSS v4)

## Goals

1. **Preserve functionality** - All existing features must continue working
2. **Gradual migration** - Migrate incrementally, not big-bang rewrite
3. **Improve UX** - Better navigation, layout, and visual hierarchy
4. **Accessibility** - WCAG 2.1 AA compliance
5. **Maintainability** - Reduce custom CSS, use composable components

## Non-Goals

1. **Backend changes** - API remains unchanged
2. **Data model changes** - No changes to storage or analysis logic
3. **New features** - Focus on UI migration, not new capabilities
4. **Framework migration** - Stay with React 18+ and Vite

## Approach

### Phase 1: Foundation (Design System Setup)
- Install shadcn-ui dependencies (Tailwind CSS v4, RadixUI primitives)
- Configure TypeScript migration path (gradual .jsx → .tsx)
- Set up theme system (CSS variables, dark mode)
- Create base layout components

### Phase 2: Core Components (Layout & Navigation)
- Implement sidebar navigation (collapsible, responsive)
- Migrate header with theme switcher
- Create main content layout wrapper
- Add breadcrumb navigation for deep views

### Phase 3: Data Display (Tables & Cards)
- Migrate metrics/traces/logs tables to shadcn Table
- Convert stat cards to shadcn Card components
- Implement search and filter components
- Add pagination components

### Phase 4: Interactive Components
- Migrate modal dialogs (Details view, confirmations)
- Implement drawer components for deep-dive views
- Add loading states and skeletons
- Implement error boundaries

### Phase 5: Polish & Accessibility
- Add keyboard shortcuts
- Implement command palette (⌘K search)
- ARIA labels and screen reader support
- Mobile responsive improvements

## Success Criteria

1. ✅ All 14 existing views work identically
2. ✅ Dark mode works across all components
3. ✅ Lighthouse accessibility score ≥90
4. ✅ No regressions in API integration
5. ✅ Build size ≤ current + 150KB (gzipped)
6. ✅ Mobile viewport support (≥375px width)

## Risks & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Bundle size increase** | High | Tree-shaking, lazy loading, measure before/after |
| **Breaking existing flows** | High | Incremental migration, feature flags for new UI |
| **Accessibility regressions** | Medium | Automated a11y tests, manual testing |
| **TypeScript complexity** | Low | Gradual migration, allow .jsx alongside .tsx |
| **Tailwind CSS learning curve** | Low | Use shadcn components as-is, minimal custom classes |

## Dependencies

- **Upstream:** None (UI-only change)
- **Downstream:** None (backend unchanged)
- **External:** 
  - shadcn-ui component library
  - Tailwind CSS v4
  - RadixUI primitives (@radix-ui/react-*)
  - class-variance-authority (cva)
  - clsx + tailwind-merge

## Timeline Estimate

- Phase 1 (Foundation): 1-2 days
- Phase 2 (Layout): 2-3 days  
- Phase 3 (Tables): 3-4 days
- Phase 4 (Interactive): 2-3 days
- Phase 5 (Polish): 1-2 days

**Total:** 9-14 days for full migration

## Alternative Approaches Considered

### Option A: Full Rewrite with shadcn-admin Template
**Pros:** Clean slate, all patterns from template  
**Cons:** High risk, lose existing work, all-or-nothing  
**Decision:** Rejected - too risky for production tool

### Option B: Keep Custom CSS, Add Component Library
**Pros:** Minimal changes  
**Cons:** Doesn't solve consistency/maintainability issues  
**Decision:** Rejected - doesn't address root problems

### Option C: Gradual Migration (Selected)
**Pros:** Low risk, incremental validation, preserve functionality  
**Cons:** Takes longer, temporary inconsistency during migration  
**Decision:** Selected - best balance of risk/reward

## Related Changes

- None (first UI modernization effort)

## Open Questions

1. **TypeScript timeline?** - Propose .jsx for now, .tsx when converting components
2. **Component library ownership?** - Copy shadcn components to `src/components/ui/` (standard practice)
3. **CSS migration strategy?** - Keep `index.css` for globals, move component styles to Tailwind
4. **Router upgrade?** - Stay with simple React state, or adopt TanStack Router?

## Feedback & Iteration

- [ ] Review with team
- [ ] Validate bundle size impact
- [ ] Test dark mode across all views
- [ ] Accessibility audit
