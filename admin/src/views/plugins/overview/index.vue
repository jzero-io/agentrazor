<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { $t } from '@/locales';
import { useRouteStore } from '@/store/modules/route';

defineOptions({ name: 'PluginOverview' });

type PluginCard = {
  menu: App.Global.Menu;
  target: App.Global.Menu | null;
};

const router = useRouter();
const routeStore = useRouteStore();

function findFirstPage(menu: App.Global.Menu): App.Global.Menu | null {
  if (menu.key && menu.routePath) return menu;

  for (const child of menu.children ?? []) {
    const target = findFirstPage(child);
    if (target) return target;
  }

  return null;
}

const plugins = computed<PluginCard[]>(() => {
  const pluginManagement = routeStore.menus.find(menu => ['plugin_management', 'plugins'].includes(menu.key));

  return (pluginManagement?.children ?? [])
    .filter(menu => menu.key !== 'plugins_overview')
    .map(menu => ({
      menu,
      target: findFirstPage(menu)
    }));
});

async function openPlugin(plugin: PluginCard) {
  if (!plugin.target) return;

  await router.push({ name: plugin.target.key });
}
</script>

<template>
  <NCard :title="$t('page.pluginOverview.title')" :bordered="false" size="small" class="card-wrapper">
    <NEmpty v-if="plugins.length === 0" :description="$t('page.pluginOverview.empty')" class="py-120px" />

    <NGrid v-else cols="1 s:2 m:3 l:4" responsive="screen" :x-gap="16" :y-gap="16">
      <NGi v-for="(plugin, index) in plugins" :key="plugin.menu.key || `${plugin.menu.label}-${index}`">
        <NCard :bordered="true" size="small" hoverable class="plugin-card">
          <div class="plugin-summary">
            <div class="plugin-icon" aria-hidden="true">
              <SvgIcon icon="carbon:plug" />
            </div>
            <div class="min-w-0 flex-1">
              <div class="plugin-name">{{ plugin.menu.label }}</div>
            </div>
          </div>

          <template #footer>
            <NButton type="primary" secondary block :disabled="!plugin.target" @click="openPlugin(plugin)">
              {{ plugin.target ? $t('page.pluginOverview.open') : $t('page.pluginOverview.unavailable') }}
            </NButton>
          </template>
        </NCard>
      </NGi>
    </NGrid>
  </NCard>
</template>

<style scoped>
.plugin-card {
  height: 100%;
}

.plugin-summary {
  display: flex;
  min-height: 72px;
  align-items: center;
  gap: 14px;
}

.plugin-icon {
  display: grid;
  width: 46px;
  height: 46px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 12px;
  color: rgb(var(--primary-color));
  background: color-mix(in srgb, rgb(var(--primary-color)) 12%, transparent);
  font-size: 25px;
}

.plugin-name {
  overflow: hidden;
  color: rgb(var(--base-text-color));
  font-size: 16px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
