FROM nginx:1.27-alpine

# Serve the mockup as the site root
COPY public/ /usr/share/nginx/html/

EXPOSE 80
