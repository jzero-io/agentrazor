<script setup lang="ts">
import ConversationContent from './components/ConversationContent.vue';

defineOptions({
  name: 'ConversationView',
  inheritAttrs: false
});

const ConversationContentView = ConversationContent as unknown;
</script>

<template>
  <component :is="ConversationContentView" v-bind="$attrs" />
</template>
