import type { Provider } from "../types";
import { get } from "svelte/store";
import { pocketbase } from "$lib/stores/pocketbase";

export interface DigitalOceanData {
  regions: { id: string; name: string }[];
  projects: { id: string; name: string }[];
  sizes?: Array<{
    slug: string;
    memory: number;
    vcpus: number;
    disk: number;
    transfer: number;
    price_monthly: number;
    category: string;
  }>;
}

export async function fetchDigitalOceanData(
  provider: Provider,
): Promise<DigitalOceanData> {
  const baseUrl = `${import.meta.env.VITE_API_BASE_URL}/api/providers/digitalocean`;
  const headers = {
    Authorization: `Bearer ${get(pocketbase).authStore.token}`,
    "Content-Type": "application/json",
  };

  try {
    // Fetch regions, projects, and sizes concurrently
    const [regionsResponse, projectsResponse, sizesResponse] =
      await Promise.all([
        fetch(`${baseUrl}/regions?providerId=${provider.id}`, { headers }),
        fetch(`${baseUrl}/projects?providerId=${provider.id}`, { headers }),
        provider.settings.region
          ? fetch(
              `${baseUrl}/sizes?providerId=${provider.id}&region=${provider.settings.region}`,
              { headers },
            )
          : Promise.resolve(null),
      ]);

    if (!regionsResponse.ok) {
      throw new Error(
        `Error fetching regions: ${await regionsResponse.text()}`,
      );
    }
    if (!projectsResponse.ok) {
      throw new Error(
        `Error fetching projects: ${await projectsResponse.text()}`,
      );
    }

    const regions = await regionsResponse.json();
    const projects = await projectsResponse.json();
    let sizes = [];

    if (sizesResponse && sizesResponse.ok) {
      const sizesData = await sizesResponse.json();
      sizes = sizesData
        .map((size: any) => ({
          ...size,
          category: getCategoryFromSlug(size.slug),
        }))
        .sort((a: any, b: any) => {
          // First sort by category
          const categoryOrder = ["basic", "general", "cpu", "memory"];
          const categoryDiff =
            categoryOrder.indexOf(a.category) -
            categoryOrder.indexOf(b.category);
          if (categoryDiff !== 0) return categoryDiff;
          // Then sort by price
          return a.price_monthly - b.price_monthly;
        });
    }

    return {
      regions: regions.map((r: any) => ({ id: r.slug, name: r.name })),
      projects: projects.map((p: any) => ({ id: p.id, name: p.name })),
      sizes,
    };
  } catch (error: any) {
    console.error("Error fetching DigitalOcean data:", error);
    throw new Error(`Failed to fetch DigitalOcean data: ${error.message}`);
  }
}

function getCategoryFromSlug(slug: string): string {
  if (slug.startsWith("s-")) return "basic";
  if (slug.startsWith("g-")) return "general";
  if (slug.startsWith("c-")) return "cpu";
  if (slug.startsWith("m-")) return "memory";
  return "basic";
}
