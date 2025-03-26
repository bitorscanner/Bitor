/**
 * Helper function to add the user ID filter to the URL parameters
 * 
 * @param params The existing URL parameters
 * @param showMyFindingsOnly Whether to show only the current user's findings
 * @param userId The current user's ID
 * @returns The updated URLSearchParams object
 */
export function addUserFilter(
  params: URLSearchParams,
  showMyFindingsOnly: boolean,
  userId: string
): URLSearchParams {
  if (showMyFindingsOnly && userId) {
    params.append('created_by', userId);
  }
  return params;
}

/**
 * Updates an existing cache key with the user filter
 * 
 * @param baseKey The existing cache key
 * @param showMyFindingsOnly Whether to show only the current user's findings
 * @returns The updated cache key
 */
export function updateCacheKey(
  baseKey: string,
  showMyFindingsOnly: boolean
): string {
  return `${baseKey}-myFindings:${showMyFindingsOnly}`;
} 