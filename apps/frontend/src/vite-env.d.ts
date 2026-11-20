declare module 'vue' {
  export interface Ref<T = any> {
    value: T
  }
  export interface ComputedRef<T = any> {
    readonly value: T
  }
  export type DefineComponent<P = {}, B = {}, D = any> = any

  export function ref<T>(value: T): Ref<T>
  export function ref<T = any>(): Ref<T | undefined>
  export function computed<T>(getter: () => T): ComputedRef<T>
  export function onMounted(hook: () => any): void
  export function onBeforeUnmount(hook: () => any): void
  export function createApp(rootComponent: any): {
    mount(rootContainer: Element | string): any
  }
}

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL?: string
  readonly VITE_MAPBOX_TOKEN?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare module 'node:test' {
  export function describe(name: string, fn: () => void | Promise<void>): void
  export function it(name: string, fn: () => void | Promise<void>): void
}

declare module 'node:assert/strict' {
  const assert: {
    equal<T>(actual: T, expected: T, message?: string): void
    deepEqual<T>(actual: T, expected: T, message?: string): void
    ok(value: unknown, message?: string): void
  }
  export default assert
}

