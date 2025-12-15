/// <reference types="vite/client" />

declare module '*.mdx' {
    import type { ComponentType, MDXProps } from 'react';
    const component: ComponentType<MDXProps>;
    export default component;
}
