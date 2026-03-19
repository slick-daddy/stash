// OCounter resolver method
func (r *Resolver) OCounter(galleryID int) (int, error) {
    return r.repository.Image.OCountByGalleryID(galleryID)
}