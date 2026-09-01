import type { App } from 'vue';
import { createRouter, createWebHistory } from 'vue-router';
import { createBuiltinRoutes } from './routes/builtin';

export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: createBuiltinRoutes()
});

export async function setupRouter(app: App) {
  app.use(router);
  await router.isReady();
}
