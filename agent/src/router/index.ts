import type { App } from 'vue';
import { createRouter, createWebHistory } from 'vue-router';
import { createBuiltinRoutes } from './routes/builtin';

const ADMIN_MOUNT_PATH = '/agent-app';

export function getRouterBase(pathname = window.location.pathname) {
  return pathname === ADMIN_MOUNT_PATH || pathname.startsWith(`${ADMIN_MOUNT_PATH}/`)
    ? `${ADMIN_MOUNT_PATH}/`
    : '/';
}

export const router = createRouter({
  history: createWebHistory(getRouterBase()),
  routes: createBuiltinRoutes()
});

export async function setupRouter(app: App) {
  app.use(router);
  await router.isReady();
}
