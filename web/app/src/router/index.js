import {createRouter, createWebHistory} from 'vue-router'
import Home from '@/views/Home'
import EndpointDetails from "@/views/EndpointDetails";
import SuiteDetails from '@/views/SuiteDetails';
import AdminEndpoints from '@/views/admin/AdminEndpoints';
import AdminEndpointForm from '@/views/admin/AdminEndpointForm';

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
    }
];

const router = createRouter({
    history: createWebHistory(process.env.BASE_URL),
    routes
});

export default router;
