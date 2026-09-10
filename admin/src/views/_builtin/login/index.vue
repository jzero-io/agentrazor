<script setup lang="ts">
import { computed } from 'vue';
import type { Component } from 'vue';
import { $t } from '@/locales';
import { useAppStore } from '@/store/modules/app';
import { useThemeStore } from '@/store/modules/theme';
import { loginModuleRecord } from '@/constants/app';
import PwdLogin from './modules/pwd-login.vue';
import CodeLogin from './modules/code-login.vue';
import Register from './modules/register.vue';
import ResetPwd from './modules/reset-pwd.vue';
import BindWechat from './modules/bind-wechat.vue';

interface Props {
  /** The login module */
  module?: UnionKey.LoginModule;
}

const props = defineProps<Props>();

const appStore = useAppStore();
const themeStore = useThemeStore();

interface LoginModule {
  label: string;
  component: Component;
}

const moduleMap: Record<UnionKey.LoginModule, LoginModule> = {
  'pwd-login': { label: loginModuleRecord['pwd-login'], component: PwdLogin },
  'code-login': { label: loginModuleRecord['code-login'], component: CodeLogin },
  register: { label: loginModuleRecord.register, component: Register },
  'reset-pwd': { label: loginModuleRecord['reset-pwd'], component: ResetPwd },
  'bind-wechat': { label: loginModuleRecord['bind-wechat'], component: BindWechat }
};

const activeModule = computed(() => moduleMap[props.module || 'pwd-login']);
</script>

<template>
  <div class="login-page" :class="{ 'login-page--dark': themeStore.darkMode }">
    <div class="login-background" aria-hidden="true">
      <div class="login-background__grid"></div>
      <div class="login-background__glow login-background__glow--top"></div>
      <div class="login-background__glow login-background__glow--bottom"></div>

      <svg class="login-route-map" viewBox="0 0 1440 900" preserveAspectRatio="xMidYMid slice">
        <g class="login-route-map__paths">
          <path d="M-70 228 132 228 238 122 455 122 558 225" />
          <path d="M-40 695 174 695 305 564 470 564 574 460" />
          <path d="M1015 82 1119 186 1302 186 1490 374" />
          <path d="M916 680 1045 551 1208 551 1486 273" />
          <path d="M1058 792 1185 665 1458 665" />
        </g>
        <g class="login-route-map__nodes">
          <circle cx="132" cy="228" r="7" />
          <circle cx="238" cy="122" r="5" />
          <circle cx="305" cy="564" r="7" />
          <circle cx="470" cy="564" r="5" />
          <circle cx="1119" cy="186" r="7" />
          <circle cx="1208" cy="551" r="5" />
          <circle cx="1185" cy="665" r="7" />
        </g>
        <g class="login-route-map__signals">
          <circle cx="455" cy="122" r="4" />
          <circle cx="174" cy="695" r="4" />
          <circle cx="1302" cy="186" r="4" />
          <circle cx="1045" cy="551" r="4" />
        </g>
      </svg>

      <div class="login-background__mark login-background__mark--left"></div>
      <div class="login-background__mark login-background__mark--right"></div>
    </div>

    <NCard :bordered="false" class="login-card relative z-4 w-auto rd-12px">
      <div class="w-400px lt-sm:w-300px">
        <header class="flex-y-center justify-between">
          <SystemLogo class="text-64px text-primary lt-sm:text-48px" />
          <h3 class="text-28px text-base-text font-500 lt-sm:text-22px">{{ $t('system.title') }}</h3>
          <div class="i-flex-col">
            <ThemeSchemaSwitch
              :theme-schema="themeStore.themeScheme"
              :show-tooltip="false"
              class="text-20px lt-sm:text-18px"
              @switch="themeStore.toggleThemeScheme"
            />
            <LangSwitch
              :lang="appStore.locale"
              :lang-options="appStore.localeOptions"
              :show-tooltip="false"
              @change-lang="appStore.changeLocale"
            />
          </div>
        </header>
        <main class="pt-24px">
          <h3 class="text-18px text-primary font-medium">{{ $t(activeModule.label) }}</h3>
          <div class="pt-24px">
            <Transition :name="themeStore.page.animateMode" mode="out-in" appear>
              <component :is="activeModule.component" />
            </Transition>
          </div>
        </main>
      </div>
    </NCard>
  </div>
</template>

<style scoped>
.login-page {
  --login-accent: #2e82c3;
  --login-accent-soft: #78bceb;
  --login-signal: #f4a340;
  position: relative;
  display: flex;
  width: 100%;
  height: 100%;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  background: #f4f8fa;
  isolation: isolate;
}

.login-background {
  position: absolute;
  inset: 0;
  z-index: 0;
  overflow: hidden;
  pointer-events: none;
}

.login-background::before {
  position: absolute;
  inset: 0;
  background: radial-gradient(
      circle at 50% 48%,
      rgb(255 255 255 / 92%) 0,
      rgb(255 255 255 / 58%) 240px,
      transparent 540px
    ),
    linear-gradient(135deg, rgb(221 238 246 / 82%), transparent 44%, rgb(229 238 242 / 68%));
  content: '';
}

.login-background__grid {
  position: absolute;
  inset: 0;
  background-image: linear-gradient(rgb(73 126 151 / 7%) 1px, transparent 1px),
    linear-gradient(90deg, rgb(73 126 151 / 7%) 1px, transparent 1px);
  background-size: 48px 48px;
  mask-image: radial-gradient(ellipse at center, transparent 6%, #000 72%);
}

.login-background__glow {
  position: absolute;
  width: 520px;
  height: 520px;
  border-radius: 50%;
  filter: blur(2px);
  opacity: 0.72;
}

.login-background__glow--top {
  top: -330px;
  right: -150px;
  background: radial-gradient(circle, rgb(46 130 195 / 24%), rgb(46 130 195 / 7%) 55%, transparent 72%);
}

.login-background__glow--bottom {
  bottom: -350px;
  left: -120px;
  background: radial-gradient(circle, rgb(104 174 216 / 28%), rgb(104 174 216 / 8%) 56%, transparent 72%);
}

.login-route-map {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  opacity: 0.76;
}

.login-route-map__paths {
  fill: none;
  stroke: var(--login-accent);
  stroke-dasharray: 4 9;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 1.5;
  animation: route-flow 28s linear infinite;
}

.login-route-map__nodes circle {
  fill: #f8fcfe;
  stroke: var(--login-accent);
  stroke-width: 3;
  filter: drop-shadow(0 0 8px rgb(46 130 195 / 38%));
}

.login-route-map__signals circle {
  fill: var(--login-signal);
  filter: drop-shadow(0 0 7px rgb(244 163 64 / 52%));
}

.login-background__mark {
  position: absolute;
  width: 220px;
  height: 42px;
  border: 1px solid rgb(46 130 195 / 15%);
  background: linear-gradient(90deg, transparent, rgb(46 130 195 / 8%), transparent);
  transform: rotate(-45deg);
}

.login-background__mark--left {
  bottom: 88px;
  left: 5%;
}

.login-background__mark--right {
  top: 110px;
  right: 8%;
}

.login-card {
  border: 1px solid rgb(196 216 225 / 78%);
  background: rgb(255 255 255 / 88%);
  box-shadow:
    0 24px 70px rgb(45 86 105 / 14%),
    0 2px 8px rgb(45 86 105 / 6%);
  backdrop-filter: blur(22px) saturate(118%);
}

.login-page--dark {
  background: #181a1b;
}

.login-page--dark .login-background::before {
  background: radial-gradient(circle at 50% 48%, rgb(38 44 47 / 86%) 0, rgb(28 31 33 / 68%) 270px, transparent 560px),
    linear-gradient(135deg, rgb(28 43 50 / 90%), transparent 48%, rgb(24 30 33 / 82%));
}

.login-page--dark .login-background__grid {
  background-image: linear-gradient(rgb(131 192 235 / 7%) 1px, transparent 1px),
    linear-gradient(90deg, rgb(131 192 235 / 7%) 1px, transparent 1px);
}

.login-page--dark .login-route-map__nodes circle {
  fill: #202426;
  stroke: var(--login-accent-soft);
}

.login-page--dark .login-card {
  border-color: rgb(84 104 113 / 52%);
  background: rgb(31 34 36 / 88%);
  box-shadow:
    0 28px 80px rgb(0 0 0 / 32%),
    0 2px 10px rgb(0 0 0 / 22%);
}

@keyframes route-flow {
  to {
    stroke-dashoffset: -130;
  }
}

@media (max-width: 640px) {
  .login-route-map {
    width: 160%;
    transform: translateX(-18%);
    opacity: 0.48;
  }

  .login-background__grid {
    background-size: 36px 36px;
  }

  .login-card {
    max-width: calc(100% - 32px);
  }
}

@media (prefers-reduced-motion: reduce) {
  .login-route-map__paths {
    animation: none;
  }
}

:deep(.login-form .n-input) {
  box-shadow: none !important;
}

:deep(.login-form .n-input-wrapper) {
  background: transparent !important;
  box-shadow: none !important;
}

:deep(.login-form .n-input__input-el) {
  background: transparent !important;
  box-shadow: none !important;
}

:deep(.login-form input:-webkit-autofill),
:deep(.login-form input:-webkit-autofill:hover),
:deep(.login-form input:-webkit-autofill:focus),
:deep(.login-form input:-webkit-autofill:active) {
  background-color: transparent !important;
  background-image: none !important;
  box-shadow: none !important;
  -webkit-background-clip: text !important;
  background-clip: text !important;
  -webkit-text-fill-color: var(--n-text-color) !important;
  caret-color: var(--n-text-color);
  transition: background-color 999999s ease-out 0s;
}
</style>
