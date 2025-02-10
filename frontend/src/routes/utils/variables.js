export const imgDir = import.meta.env.VITE_API_BASE_URL;

/** @type {(userId: string, avatarName: string) => string} */
export const avatarPath = (userId, avatarName) =>
  `${imgDir}/api/files/users/${userId}/${avatarName}`;

/** @type {(x:string, ...y:string[]) => string} */
export const imagesPath = (src, ...subdirs) =>
  [imgDir, ...subdirs, src].filter(Boolean).join("/");

// Add a default avatar path
export const defaultAvatarPath = "/images/default-avatar.png";
