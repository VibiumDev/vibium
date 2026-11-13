/** Make an existing session object callable without creating another session. */
export function callable<T extends { run: (...args: any[]) => any }>(instance: T): T {
  const bound = new Map<PropertyKey, { original: Function; value: Function }>();
  const invoke = (...args: Parameters<T['run']>): ReturnType<T['run']> => instance.run(...args);
  Object.setPrototypeOf(invoke, Object.getPrototypeOf(instance));
  const proxy = new Proxy(invoke, {
    get(target, key, receiver) {
      if (!(key in instance)) return Reflect.get(target, key, receiver);
      const value = Reflect.get(instance, key, instance);
      // Instance-owned callable namespaces (waitUntil, capture, etc.)
      // already close over their session and carry their own properties.
      if (typeof value !== 'function' || key === 'constructor' || Object.prototype.hasOwnProperty.call(instance, key)) return value;
      const cached = bound.get(key);
      if (cached?.original === value) return cached.value;
      const method = (...args: unknown[]) => {
        const result = value.apply(instance, args);
        if (result === instance) return proxy;
        if (result instanceof Promise) {
          const mapped = result.then(v => v === instance ? proxy : v);
          // Fluent find() promises carry action methods of their own.
          Object.defineProperties(mapped, Object.getOwnPropertyDescriptors(result));
          return mapped;
        }
        return result;
      };
      bound.set(key, { original: value, value: method });
      return method;
    },
    set(_target, key, value) { return Reflect.set(instance, key, value, instance); },
    has(target, key) { return key in instance || key in target; },
  }) as unknown as T;
  return proxy;
}
