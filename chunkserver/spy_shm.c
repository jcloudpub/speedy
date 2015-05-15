#include "spy_shm.h"

char* spy_create_and_attach_shm(key_t shm_key, size_t shm_size)
{
	int shm_id;
	char *shm_ptr;

	shm_id = shmget(shm_key, shm_size, IPC_CREAT | IPC_EXCL | 0666);

	if (shm_id < 0) {
		if (errno != EEXIST) {			
			return NULL;
		}

		shm_id = shmget(shm_key, shm_size, 0666);
		if (shm_id < 0) {
			if ((shm_id = shmget(shm_key, 0, 0666)) < 0 || 
				shmctl(shm_id, IPC_RMID, NULL)) {
				return NULL;
			}
			
			shm_id = shmget(shm_key, shm_size, IPC_CREAT | IPC_EXCL | 0666);
			if (shm_id < 0) {
				return NULL;
			}
		}
	}

	//if shall not be attached, shmat() shall return -1	
	shm_ptr = (char *)shmat(shm_id, NULL, 0);
	if (shm_ptr == (char*)-1) {		
		return NULL;
	}
	
	return shm_ptr;
}
