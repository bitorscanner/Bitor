/**
 * Generates a UUID v4 string
 * Uses crypto.randomUUID() if available, otherwise falls back to a polyfill
 */
export function generateUUID(): string {
  // Try to use the native crypto.randomUUID() if available
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    try {
      return crypto.randomUUID();
    } catch (error) {
      console.warn('crypto.randomUUID() failed, falling back to polyfill:', error);
    }
  }

  // Fallback for older browsers or non-secure contexts
  return uuidv4Polyfill();
}

/**
 * Polyfill implementation of UUID v4 generation
 */
function uuidv4Polyfill(): string {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
    const r = Math.random() * 16 | 0;
    const v = c === 'x' ? r : (r & 0x3 | 0x8);
    return v.toString(16);
  });
}

// Alias for backwards compatibility
export const randomUUID = generateUUID; 