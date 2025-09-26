# Roadmap / Features a futuro

## 🎯 Corto plazo

- [ ] Adjuntar **múltiples archivos** en un solo webhook (imagen + history.txt en la misma request).
- [ ] Persistencia del historial en **disco/DB** además de memoria (resiliencia a reinicios).
- [ ] Endpoint `/healthz` y `/readyz`.
- [ ] Logs estructurados (zap/logrus) con **correlation IDs** por mensaje.
- [ ] Configurable **retención** de historial por JID (TTL / tamaño máximo).

## 🚀 Mediano plazo

- [ ] UI mínima (panel web) para listar sesiones/devices y su estado.
- [ ] **Reacciones** o comandos desde Discord → acciones en WhatsApp (p.ej. enviar respuesta).
- [ ] Soporte de **media** adicional: audio, video, docs (subida a S3 y link en Discord).
- [ ] Multi-webhook (ruteo por JID: distintos canales de Discord para distintos números).
- [ ] Filtrado/normalización de nombres de archivo y sanitización avanzada.

## 🧠 Largo plazo

- [ ] Clasificación de mensajes con un **pipeline** (spam/soporte/ventas) y ruteo automático.
- [ ] Búsqueda full-text de historial (SQLite/PG + FTS).
- [ ] Enriquecimiento: detección de idioma, entidades, links.
- [ ] Integración con **tickets** (Jira/Linear) desde reacciones en Discord.
- [ ] Exportación de conversación a **PDF** y envío automático.
