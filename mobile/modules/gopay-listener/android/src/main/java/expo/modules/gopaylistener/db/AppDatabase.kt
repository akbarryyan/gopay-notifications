package expo.modules.gopaylistener.db

import android.content.Context
import androidx.room.Database
import androidx.room.Room
import androidx.room.RoomDatabase
import androidx.room.TypeConverters

@Database(entities = [EventEntity::class], version = 1, exportSchema = false)
@TypeConverters(EventStatusConverter::class)
abstract class AppDatabase : RoomDatabase() {

    abstract fun events(): EventDao

    companion object {
        @Volatile
        private var instance: AppDatabase? = null

        /**
         * Database dimiliki lapisan native. React Native tidak pernah membuka
         * berkas ini — dua penulis ke satu berkas adalah sumber bug.
         */
        fun get(context: Context): AppDatabase =
            instance ?: synchronized(this) {
                instance ?: Room.databaseBuilder(
                    context.applicationContext,
                    AppDatabase::class.java,
                    "gopay-bridge.db"
                ).build().also { instance = it }
            }
    }
}
