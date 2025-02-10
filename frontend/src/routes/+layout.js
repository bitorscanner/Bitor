/** @type {import('./$types').LayoutLoad} */
export async function load({ fetch }) {
  // Server-side load function should just return empty data
  // The actual user data loading will happen in the layout.svelte component
  return {
    user: null,
  };
}

export const prerender = true;
export const trailingSlash = "always";
