export async function getFavicon(url: string): Promise<string | null> {
  try {
    console.log("Getting favicon for URL:", url);
    // Remove protocol and get domain
    const domain = url.replace(/^(https?:\/\/)?(www\.)?/, "").split("/")[0];
    console.log("Extracted domain:", domain);

    // Try Google Favicon service first
    const googleFaviconUrl = `https://www.google.com/s2/favicons?domain=${domain}&sz=64`;
    console.log("Generated Google Favicon URL:", googleFaviconUrl);

    // Return the Google Favicon URL
    return googleFaviconUrl;
  } catch (error) {
    console.error("Error fetching favicon:", error);
    return null;
  }
}
