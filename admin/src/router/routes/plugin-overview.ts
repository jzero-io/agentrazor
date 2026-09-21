import type { ElegantConstRoute } from '@elegant-router/types';

const PLUGIN_MANAGEMENT_ROUTE = 'plugin_management';
const PLUGIN_OVERVIEW_ROUTE = 'plugins_overview';

function createPluginOverviewRoute(): ElegantConstRoute {
  return {
    name: PLUGIN_OVERVIEW_ROUTE,
    path: '/plugins/overview',
    component: 'view.plugins_overview',
    meta: {
      title: PLUGIN_OVERVIEW_ROUTE,
      i18nKey: 'route.plugins_overview',
      icon: 'carbon:catalog',
      order: 0
    }
  };
}

/**
 * Add the platform landing page to an authorized Plugin Management route.
 *
 * The server already filters the returned route tree by role. Keeping this page client-owned avoids creating a separate
 * database permission merely to render the empty state and the current role's authorized plugin entries.
 */
export function withPluginOverview(routes: ElegantConstRoute[]): ElegantConstRoute[] {
  return routes.map(route => {
    if (route.name !== PLUGIN_MANAGEMENT_ROUTE) return route;
    if (route.children?.some(child => child.name === PLUGIN_OVERVIEW_ROUTE)) return route;

    return {
      ...route,
      children: [createPluginOverviewRoute(), ...(route.children ?? [])]
    };
  });
}
