import { pocketbase } from '@lib/stores/pocketbase';
import { get } from 'svelte/store';

export async function getFavicon(url: string, clientId?: string): Promise<string | null> {
  try {
    console.log("Getting favicon for URL:", url);
    // Remove protocol and get domain
    const domain = url.replace(/^(https?:\/\/)?(www\.)?/, "").split("/")[0];
    console.log("Extracted domain:", domain);

    if (clientId) {
      // Use the fetch-favicon endpoint to save the favicon
      const token = get(pocketbase).authStore.token;  // Get the auth token from the pocketbase client
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/clients/${clientId}/fetch-favicon`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (!response.ok) {
        console.error("Failed to fetch favicon:", response.statusText);
        return null;
      }

      const data = await response.json();
      return data.favicon; // This will be a data URI
    } else {
      // Use the proxy endpoint for display only
      return `${import.meta.env.VITE_API_BASE_URL}/api/clients/favicon?domain=${encodeURIComponent(domain)}`;
    }
  } catch (error) {
    console.error("Error fetching favicon:", error);
    return null;
  }
}
