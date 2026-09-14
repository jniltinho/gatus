<template>
  <div
    v-if="open"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 dark:bg-black/70 p-4"
    role="dialog"
    aria-modal="true"
    data-testid="confirm-dialog"
    @keydown.esc="$emit('cancel')"
  >
    <div class="w-full max-w-md border bg-card text-card-foreground shadow-lg dark:border-gray-700 dark:bg-gray-900">
      <div class="px-6 pt-5 pb-2">
        <h2 class="text-lg font-semibold text-foreground dark:text-gray-100">{{ title }}</h2>
      </div>
      <p class="px-6 pb-4 text-sm text-muted-foreground dark:text-gray-400 whitespace-pre-line">{{ message }}</p>
      <div class="flex justify-end gap-2 px-6 pb-5">
        <Button variant="outline" data-testid="confirm-cancel" @click="$emit('cancel')">Cancelar</Button>
        <Button variant="destructive" data-testid="confirm-accept" @click="$emit('confirm')">{{ confirmLabel }}</Button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { Button } from '@/components/ui/button'

defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, required: true },
  message: { type: String, required: true },
  confirmLabel: { type: String, default: 'Confirmar' },
})

defineEmits(['confirm', 'cancel'])
</script>
