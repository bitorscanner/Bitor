import { redirect } from "@sveltejs/kit";
import { pocketbase } from "$lib/stores/pocketbase";
import { currentUser } from "$lib/stores/auth";
import { get } from "svelte/store";
import { browser } from "$app/environment";

const permissionMap: Record<string, string> = {
  "/settings/users": "manage_users",
  "/settings/groups": "manage_groups",
  "/settings/providers": "manage_providers",
  "/settings/notifications": "manage_notifications",
  "/settings/system": "manage_system",
};

export const load = async ({ url }) => {
  // Skip all checks if we're not in the browser
  if (!browser) {
    return {};
  }

  const path = url.pathname;
  const pb = get(pocketbase);

  // Account settings is always accessible
  if (path === "/settings/account") {
    return {};
  }

  // Check if we have a valid auth session
  if (!pb?.authStore?.isValid) {
    throw redirect(303, "/authentication/sign-in");
  }

  // If user is a PocketBase admin, they have access to everything
  if (pb.authStore.isAdmin) {
    return {};
  }

  const user = get(currentUser);

  // If user is not loaded, try to load it
  if (!user) {
    try {
      if (!pb.authStore.model?.id) {
        throw new Error("No user ID found in auth store");
      }
      const userData = await pb
        .collection("users")
        .getOne(pb.authStore.model.id, {
          expand: "group",
        });
      currentUser.set(userData);

      // Admin group members have access to everything
      if (userData?.expand?.group?.name === "admin") {
        return {};
      }

      // Check specific permissions
      const requiredPermission = permissionMap[path];
      if (requiredPermission) {
        const hasPermission =
          userData?.expand?.group?.permissions?.[requiredPermission] === true;
        if (!hasPermission) {
          throw redirect(303, "/settings");
        }
      }
    } catch (error) {
      console.error("Error loading user data:", error);
      throw redirect(303, "/authentication/sign-in");
    }
  } else {
    // User is already loaded, check permissions
    if (user?.group?.name === "admin") {
      return {};
    }

    // Check specific permissions
    const requiredPermission = permissionMap[path];
    if (requiredPermission) {
      const hasPermission =
        user?.group?.permissions?.[requiredPermission] === true;
      if (!hasPermission) {
        throw redirect(303, "/settings");
      }
    }
  }

  return {};
};
