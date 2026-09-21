import { useTitle } from '@vueuse/core';
import type { Router } from 'vue-router';
import { translateOr } from '@/locales';

export function createDocumentTitleGuard(router: Router) {
  router.afterEach(to => {
    const { i18nKey, title } = to.meta;

    const documentTitle = translateOr(i18nKey, title);

    useTitle(documentTitle);
  });
}
