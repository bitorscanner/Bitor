const ENCRYPTION_KEY = "orbit_key_encryption";

/**
 * Encrypts sensitive data using AES-GCM
 */
export async function encryptData(data: string): Promise<string> {
  const encoder = new TextEncoder();
  const keyMaterial = await window.crypto.subtle.importKey(
    "raw",
    encoder.encode(ENCRYPTION_KEY),
    { name: "PBKDF2" },
    false,
    ["deriveBits", "deriveKey"],
  );

  const salt = window.crypto.getRandomValues(new Uint8Array(16));
  const iv = window.crypto.getRandomValues(new Uint8Array(12));

  const key = await window.crypto.subtle.deriveKey(
    {
      name: "PBKDF2",
      salt,
      iterations: 100000,
      hash: "SHA-256",
    },
    keyMaterial,
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt"],
  );

  const encryptedContent = await window.crypto.subtle.encrypt(
    {
      name: "AES-GCM",
      iv,
    },
    key,
    encoder.encode(data),
  );

  // Combine the salt, iv, and encrypted content into a single array
  const encryptedArray = new Uint8Array(
    salt.length + iv.length + encryptedContent.byteLength,
  );
  encryptedArray.set(salt, 0);
  encryptedArray.set(iv, salt.length);
  encryptedArray.set(new Uint8Array(encryptedContent), salt.length + iv.length);

  // Convert to base64 for storage
  return btoa(String.fromCharCode(...encryptedArray));
}

/**
 * Decrypts encrypted data using AES-GCM
 */
export async function decryptData(encryptedData: string): Promise<string> {
  const decoder = new TextDecoder();
  const encoder = new TextEncoder();

  // Convert from base64 and extract salt, iv, and encrypted content
  const encryptedArray = new Uint8Array(
    atob(encryptedData)
      .split("")
      .map((char) => char.charCodeAt(0)),
  );

  const salt = encryptedArray.slice(0, 16);
  const iv = encryptedArray.slice(16, 28);
  const encryptedContent = encryptedArray.slice(28);

  const keyMaterial = await window.crypto.subtle.importKey(
    "raw",
    encoder.encode(ENCRYPTION_KEY),
    { name: "PBKDF2" },
    false,
    ["deriveBits", "deriveKey"],
  );

  const key = await window.crypto.subtle.deriveKey(
    {
      name: "PBKDF2",
      salt,
      iterations: 100000,
      hash: "SHA-256",
    },
    keyMaterial,
    { name: "AES-GCM", length: 256 },
    false,
    ["decrypt"],
  );

  const decryptedContent = await window.crypto.subtle.decrypt(
    {
      name: "AES-GCM",
      iv,
    },
    key,
    encryptedContent,
  );

  return decoder.decode(decryptedContent);
}
