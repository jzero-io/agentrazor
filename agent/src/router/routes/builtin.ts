import type { RouteRecordRaw } from 'vue-router';

const ConversationLayout = () => import('../../layouts/conversation/index.vue');
const ConversationView = () => import('../../views/conversation/index.vue');
const SettingsView = () => import('../../views/settings/index.vue');

export function createBuiltinRoutes(): RouteRecordRaw[] {
  return [
    {
      path: '/',
      component: ConversationLayout,
      children: [
        {
          path: '',
          name: 'home',
          component: ConversationView
        },
        {
          path: 'c/:conversationId',
          name: 'conversation-detail',
          component: ConversationView
        },
        {
          path: 'settings/appearance',
          name: 'settings-appearance',
          component: SettingsView
        },
        {
          path: 'settings/archives',
          name: 'settings-archives',
          component: SettingsView
        },
        {
          path: 'settings',
          redirect: '/settings/appearance'
        }
      ]
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/'
    }
  ];
}
