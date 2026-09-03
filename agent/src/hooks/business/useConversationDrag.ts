import { onBeforeUnmount, onMounted, ref, type Ref } from 'vue';
import type { Conversation } from '../../service/api';
import type { ConversationGroup } from './useConversationGroups';

interface UseConversationDragOptions {
  conversationListDropTarget: string;
  conversationGroups: Ref<ConversationGroup[]>;
  conversationsExpanded: Ref<boolean>;
  displayConversationTitle: (conversation: Conversation) => string;
  hideConversationPreview: () => void;
  selectConversation: (conversation: Conversation) => void | Promise<void>;
  updateConversationGroup: (conversation: Conversation, groupId: string) => void | Promise<void>;
}

export function useConversationDrag(options: UseConversationDragOptions) {
  const draggedConversationId = ref('');
  const conversationDropTarget = ref('');
  let pointerDragStart: { item: Conversation; x: number; y: number } | null = null;
  let dragGhostEl: HTMLElement | null = null;
  let touchDrag: { item: Conversation; x: number; y: number; activated: boolean } | null = null;
  let touchLongPressTimer: number | undefined;

  function onConversationPointerDown(item: Conversation, event: PointerEvent) {
    if (event.button !== 0 || event.pointerType === 'touch') return;
    pointerDragStart = { item, x: event.clientX, y: event.clientY };
  }

  function activateDrag(item: Conversation, x: number, y: number) {
    draggedConversationId.value = item.id;
    conversationDropTarget.value = '';
    options.hideConversationPreview();
    document.body.classList.add('is-dragging');
    dragGhostEl = document.createElement('div');
    dragGhostEl.className = 'conversation-drag-ghost';
    dragGhostEl.textContent = options.displayConversationTitle(item);
    document.body.appendChild(dragGhostEl);
    updateDragGhost(x, y);
  }

  function updateDragGhost(x: number, y: number) {
    if (!dragGhostEl) return;
    dragGhostEl.style.left = `${x + 12}px`;
    dragGhostEl.style.top = `${y + 10}px`;
  }

  function updateDropTarget(x: number, y: number) {
    const zone = document.elementFromPoint(x, y)
      ?.closest?.('.conversation-drop-zone') as HTMLElement | null | undefined;
    conversationDropTarget.value = zone ? String(zone.dataset.dropTarget || zone.dataset.groupId || '') : '';
  }

  function dropTargetGroupId(target: string) {
    return target === options.conversationListDropTarget ? '' : target;
  }

  function clearDragVisuals() {
    conversationDropTarget.value = '';
    draggedConversationId.value = '';
    if (dragGhostEl) {
      dragGhostEl.remove();
      dragGhostEl = null;
    }
    document.body.classList.remove('is-dragging');
  }

  function finishDrag(item: Conversation) {
    const target = conversationDropTarget.value;
    clearDragVisuals();
    if (!target) return;

    const targetGroupId = dropTargetGroupId(target);
    if ((item.groupId || '') === targetGroupId) return;
    if (targetGroupId) {
      const group = options.conversationGroups.value.find(candidate => candidate.id === targetGroupId);
      if (group) group.collapsed = false;
    } else {
      options.conversationsExpanded.value = true;
    }
    void options.updateConversationGroup(item, targetGroupId);
  }

  function onDragPointerMove(event: PointerEvent) {
    if (!pointerDragStart) return;
    const dx = event.clientX - pointerDragStart.x;
    const dy = event.clientY - pointerDragStart.y;
    if (!draggedConversationId.value) {
      if (Math.hypot(dx, dy) < 8) return;
      activateDrag(pointerDragStart.item, event.clientX, event.clientY);
    }
    updateDragGhost(event.clientX, event.clientY);
    updateDropTarget(event.clientX, event.clientY);
  }

  function onDragPointerUp() {
    if (!pointerDragStart) return;
    const item = pointerDragStart.item;
    pointerDragStart = null;
    finishDrag(item);
  }

  function onRowTouchStart(item: Conversation, event: TouchEvent) {
    if (event.touches.length !== 1) return;
    const target = event.target;
    if (target instanceof Element && target.closest('button:not(.conversation-item)')) return;
    const touch = event.touches[0];
    touchDrag = { item, x: touch.clientX, y: touch.clientY, activated: false };
    touchLongPressTimer = window.setTimeout(() => {
      if (touchDrag && !touchDrag.activated) {
        touchDrag.activated = true;
        activateDrag(touchDrag.item, touchDrag.x, touchDrag.y);
      }
    }, 350);
  }

  function onWindowTouchMove(event: TouchEvent) {
    if (!touchDrag || event.touches.length !== 1) return;
    const touch = event.touches[0];
    const dx = touch.clientX - touchDrag.x;
    const dy = touch.clientY - touchDrag.y;
    if (!touchDrag.activated) {
      if (Math.hypot(dx, dy) > 8) {
        window.clearTimeout(touchLongPressTimer);
        touchDrag = null;
      }
      return;
    }
    event.preventDefault();
    updateDragGhost(touch.clientX, touch.clientY);
    updateDropTarget(touch.clientX, touch.clientY);
  }

  function onWindowTouchEnd(event: TouchEvent) {
    if (!touchDrag) return;
    window.clearTimeout(touchLongPressTimer);
    const item = touchDrag.item;
    const activated = touchDrag.activated;
    touchDrag = null;
    if (activated) {
      finishDrag(item);
      return;
    }

    event.preventDefault();
    options.hideConversationPreview();
    void options.selectConversation(item);
  }

  function onWindowTouchCancel() {
    window.clearTimeout(touchLongPressTimer);
    touchDrag = null;
    clearDragVisuals();
  }

  function cleanupDrag() {
    pointerDragStart = null;
    touchDrag = null;
    window.clearTimeout(touchLongPressTimer);
    clearDragVisuals();
  }

  onMounted(() => {
    window.addEventListener('pointermove', onDragPointerMove);
    window.addEventListener('pointerup', onDragPointerUp);
    window.addEventListener('touchmove', onWindowTouchMove, { passive: false });
    window.addEventListener('touchend', onWindowTouchEnd, { passive: false });
    window.addEventListener('touchcancel', onWindowTouchCancel);
  });

  onBeforeUnmount(() => {
    window.removeEventListener('pointermove', onDragPointerMove);
    window.removeEventListener('pointerup', onDragPointerUp);
    window.removeEventListener('touchmove', onWindowTouchMove);
    window.removeEventListener('touchend', onWindowTouchEnd);
    window.removeEventListener('touchcancel', onWindowTouchCancel);
    cleanupDrag();
  });

  return {
    draggedConversationId,
    conversationDropTarget,
    onConversationPointerDown,
    onRowTouchStart,
    cleanupDrag
  };
}
