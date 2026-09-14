import {createRouter, createWebHistory} from 'vue-router'
import Home from '@/views/Home'
import EndpointDetails from "@/views/EndpointDetails";
import SuiteDetails from '@/views/SuiteDetails';
import AdminEndpoints from '@/views/admin/AdminEndpoints';
import AdminEndpointForm from '@/views/admin/AdminEndpointForm';
import AdminStatusPages from '@/views/admin/AdminStatusPages';
import AdminStatusPageForm from '@/views/admin/AdminStatusPageForm';
import StatusPage from '@/views/public/StatusPage';
import StatusPageEndpoint from '@/views/public/StatusPageEndpoint';

const routes = [
    {
        path: '/',
        name: 'Home',
        component: Home
    },
    {
        path: '/endpoints/:key',
        name: 'EndpointDetails',
        component: EndpointDetails,
    },
    {
        path: '/suites/:key',
        name: 'SuiteDetails',
        component: SuiteDetails
    },
    // Administration of endpoints (fork)
    {
        path: '/admin',
        name: 'AdminEndpoints',
        component: AdminEndpoints
    },
    {
        path: '/admin/endpoints/new',
        name: 'AdminEndpointNew',
        component: AdminEndpointForm
    },
    {
        path: '/admin/endpoints/:endpointKey/edit',
        name: 'AdminEndpointEdit',
        component: AdminEndpointForm,
        props: true
    },
    // Administration of the status pages (fork)
    {
        path: '/admin/status-pages',
        name: 'AdminStatusPages',
        component: AdminStatusPages
    },
    {
        path: '/admin/status-pages/new',
        name: 'AdminStatusPageNew',
        component: AdminStatusPageForm
    },
    {
        path: '/admin/status-pages/:slug/edit',
        name: 'AdminStatusPageEdit',
        component: AdminStatusPageForm,
        props: true
    },
    // Public status pages (fork): no login screen and no call to /api/v1/config, see App.vue
    {
        path: '/status/:slug([a-z0-9-]{1,64})',
        name: 'PublicStatusPage',
        component: StatusPage,
        meta: { public: true }
    },
    {
        path: '/status/:slug([a-z0-9-]{1,64})/endpoints/:key',
        name: 'PublicStatusPageEndpoint',
        component: StatusPageEndpoint,
        meta: { public: true }
    },
    {
        path: '/status/:pathMatch(.*)*',
        name: 'PublicStatusPageNotFound',
        component: StatusPage,
        meta: { public: true }
    }
];

const router = createRouter({
    history: createWebHistory(process.env.BASE_URL),
    routes
});

export default router;
