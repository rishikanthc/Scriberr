---
trigger: always_on
---

1.  **Separate Server State from Client State.**
    Never store API data in Redux or Context. Use **TanStack Query** (or SWR) for caching and fetching, reserving **Zustand** or Context strictly for UI state (e.g., themes, modals).

2.  **Organize by Feature, Not File Type.**
    Abandon `/components` and `/hooks` folders. **Colocate** related code (api, hooks, components) into feature-specific directories (e.g., `features/user-profile`) to ensure maintainability as the codebase grows.

3.  **Derive State, Do Not Sync It.**
    Never use `useEffect` to update a state variable based on another state variable. Calculate the derived value directly in the render body to eliminate "glitches" and extra render cycles.

4.  **Preserve Referential Equality.**
    Wrap objects and functions passed to memoized children in **`useMemo`** and **`useCallback`**. Stable references prevent expensive, unnecessary re-renders of child components.

5.  **Eliminate Render-Fetch Waterfalls.**
    Do not chain data fetching (Parent fetches -> Renders -> Child fetches). Use route-level loaders or parallel queries to fetch all required data immediately.

6.  **Prefer Composition Over Prop Drilling.**
    Avoid passing data through five layers of components. Use **Component Composition** (passing components as `children` or props) to make the hierarchy flat and efficient.

7.  **Virtualize Long Lists.**
    Never render large datasets directly to the DOM. Use **TanStack Virtual** or `react-window` to render only the visible items, ensuring the browser remains responsive.

8.  **Lazy Load Routes.**
    Implement **Code Splitting** using `React.lazy` and `Suspense` for route-level components. This ensures users only download the JavaScript required for the current page.

9.  **Enforce Strict TypeScript.**
    Treat `any` as a compile error. Define explicit interfaces for all props and API responses to guarantee safe refactoring and self-documenting code.

10. **Isolate Logic in Custom Hooks.**
    Keep UI components pure (JSX only). Abstract complex state logic, side effects, and listeners into **Custom Hooks** to ensure logic is testable, reusable, and readable.