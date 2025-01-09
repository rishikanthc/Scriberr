// stores/templateStore.ts
import { writable } from 'svelte/store';
import { apiFetch } from '$lib/api';

interface Template {
 id: string;
 title: string;
 prompt: string;
 createdAt: string;
 updatedAt: string;
}

function createTemplateStore() {
 const { subscribe, set, update } = writable<Template[]>([]);

 return {
   subscribe,
   async add(template: Omit<Template, 'id' | 'createdAt' | 'updatedAt'>) {
     const response = await apiFetch('/api/templates', {
       method: 'POST',
       body: JSON.stringify(template)
     });
     const newTemplate = await response.json();
     update(templates => [newTemplate, ...templates]);
     return newTemplate;
   },

   async remove(id: string) {
     await apiFetch(`/api/templates/${id}`, { method: 'DELETE' });
     update(templates => templates.filter(t => t.id !== id));
   },

   async update(id: string, data: Partial<Template>) {
     const response = await apiFetch(`/api/templates/${id}`, {
       method: 'PATCH',
       body: JSON.stringify(data)
     });
     const updated = await response.json();
     update(templates => templates.map(t => 
       t.id === id ? updated : t
     ));
     return updated;
   },

   async refresh() {
     const response = await apiFetch('/api/templates');
     const templates = await response.json();
     set(templates);
   }
 };
}

export const templates = createTemplateStore();
